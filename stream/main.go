package stream

import (
	"context"
	"debrid_drive/chart"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"debrid_drive/stream/buffer"
)

type Stream struct {
	url    string
	size   uint64
	client *http.Client

	stopChannel chan struct{}
	waitChannel chan struct{}

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	buffer       *buffer.Buffer
	seekPosition atomic.Uint64

	chart *chart.Chart

	mu sync.RWMutex

	closed bool
}

var bufferCreateSize = uint64(1024 * 1024 * 1024 * 1)

func NewStream(url string, size uint64) *Stream {
	chart := chart.NewChart()

	// buffer := NewBuffer(0, min(size, bufferCreateSize), chart)
	buffer := buffer.NewBuffer(min(size, bufferCreateSize), 0)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			MaxIdleConnsPerHost: 1,
			Proxy:               http.ProxyFromEnvironment,
		},
		Timeout: 6 * time.Hour,
	}

	return &Stream{
		url:    url,
		size:   size,
		client: client,

		buffer: buffer,

		chart: chart,
	}
}

func (stream *Stream) startStream(seekPosition uint64) {
	stream.chart.LogStream(fmt.Sprintf("Stream started for position: %d\n", seekPosition))

	defer func() {
		stream.chart.LogStream(fmt.Sprintf("Stream closed for position: %d\n", seekPosition))
		stream.wg.Done()
	}()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	rangeHeader := fmt.Sprintf("bytes=%d-", max(seekPosition, 0))
	req, err := http.NewRequestWithContext(ctx, "GET", stream.url, nil)
	if err != nil {
		stream.chart.LogStream(fmt.Sprintf("Failed to create request: %v\n", err))
		stream.cancel()
		return
	}

	req.Header.Set("Range", rangeHeader)

	resp, err := stream.client.Do(req)
	if err != nil {
		stream.chart.LogStream(fmt.Sprintf("Failed to do request: %v\n", err))
		stream.cancel()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		stream.chart.LogStream(fmt.Sprintf("Status code: %d\n", resp.StatusCode))
		stream.cancel()
		return
	}

	chunk := make([]byte, 8192)

	var normalDelay time.Duration = 100 * time.Microsecond
	var retryDelay time.Duration = normalDelay

	timeStart := time.Now()
	totalBytesRead := 0

	for {
		select {
		case <-stream.context.Done():
			stream.chart.LogStream(fmt.Sprintf("Stream context done\n"))
			return
		case <-ctx.Done():
			stream.chart.LogStream(fmt.Sprintf("Request context done\n"))
			return
		case <-time.After(retryDelay):
		}

		bytesToOverwrite := max(stream.buffer.GetBytesToOverwriteSync(), 0)
		chunkSizeToRead := min(uint64(len(chunk)), bytesToOverwrite)

		if chunkSizeToRead == 0 {
			retryDelay = 100 * time.Millisecond // Retry after 100 milliseconds
			continue
		}

		bytesRead, err := resp.Body.Read(chunk[:chunkSizeToRead])

		if bytesRead > 0 {
			bytesRead, err := stream.buffer.Write(chunk[:bytesRead])
			if err != nil {
				stream.chart.LogStream(fmt.Sprintf("Write error %v\n", err))
				return // Crash ?
			}

			totalBytesRead += bytesRead
			retryDelay = normalDelay // Reset
		}

		elapsed := time.Since(timeStart)
		if elapsed > 0 && false {
			mbps := float64(totalBytesRead*8) / (1024 * 1024) / elapsed.Seconds() // Convert bytes to bits and to Mbps
			stream.chart.LogStream(fmt.Sprintf("Speed: %.2f MB/s @ %d\n", mbps, seekPosition))
		}

		switch {
		case err == io.ErrUnexpectedEOF:
			stream.chart.LogStream(fmt.Sprintf("Unexpected EOF, Bytes read: %d\n", bytesRead))
			return // Decide if the loop should crash or retry logic can be added.
		case err == io.EOF:
			stream.chart.LogStream(fmt.Sprintf("Read EOF, Bytes read: %d, Position %d\n", bytesRead, seekPosition))
			retryDelay = 5 * time.Second // Retry after 5 seconds
			continue                     // Handle end of file appropriately.
		case err != nil:
			stream.chart.LogStream(fmt.Sprintf("Failed to read: %v\n", err))
			retryDelay = 5 * time.Second // Retry after 5 seconds
			continue                     // Continue to retry on other errors.
		}
	}
}

func (stream *Stream) stopStream() {
	if stream.cancel == nil {
		return
	}

	stream.cancel()
	stream.wg.Wait()
}

func (stream *Stream) Read(p []byte) (int, error) {
	if stream.closed {
		return 0, fmt.Errorf("Streamer is closed")
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	seekPosition := stream.GetSeekPosition()
	requestedSize := uint64(len(p))

	if seekPosition+requestedSize >= stream.size {
		requestedSize = stream.size - seekPosition - 1
	}

	stream.checkAndStartBufferIfNeeded(seekPosition, requestedSize)

	n, err := stream.buffer.ReadAt(p, seekPosition)
	if err != nil {
		fmt.Printf("ReadAt error %v\n", err)
	}

	return n, err
}

func (stream *Stream) checkAndStartBufferIfNeeded(seekPosition uint64, requestedSize uint64) {
	if seekPosition >= stream.size {
		stream.chart.LogStream(fmt.Sprintf("Check: Seek position is at the end\n"))
		return
	}

	seekInBuffer := stream.buffer.IsPositionInBufferSync(seekPosition)

	if !seekInBuffer {
		stream.stopStream()

		context, cancel := context.WithCancel(context.Background())
		stream.context = context
		stream.cancel = cancel

		stream.wg.Add(1)

		stream.buffer.Reset(seekPosition)

		go stream.startStream(seekPosition)

		waitForSize := min(seekPosition+requestedSize, stream.size)

		stream.buffer.WaitForPositionInBuffer(waitForSize, stream.context)

		return
	}

	dataInBuffer := stream.buffer.IsPositionInBufferSync(seekPosition + requestedSize)

	if !dataInBuffer {
		waitForSize := min(seekPosition+requestedSize, stream.size)

		// stream.chart.LogStream(fmt.Sprintf("Check: Waiting for position %d\n", seekPosition+requestedSize))
		stream.buffer.WaitForPositionInBuffer(waitForSize, stream.context)
		// stream.chart.LogStream(fmt.Sprintf("Check: Position ready %d\n", seekPosition+requestedSize))
	}
}

func (stream *Stream) Seek(offset uint64, whence int) (uint64, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.closed {
		return 0, fmt.Errorf("Streamer is closed")
	}

	var newOffset uint64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		return 0, fmt.Errorf("TODO: SeekCurrent is not supported")
	case io.SeekEnd:
		return 0, fmt.Errorf("SeekEnd is not supported")
	default:
		return 0, fmt.Errorf("Invalid whence: %d", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("Negative position is invalid")
	}

	var err error
	if newOffset >= stream.size {
		newOffset = stream.size
		err = io.EOF
	}

	stream.seekPosition.Store(newOffset)

	stream.chart.UpdateSeekTotal(newOffset, stream.size)

	return newOffset, err
}

func (stream *Stream) GetSeekPosition() uint64 {
	return stream.seekPosition.Load()
}

func (stream *Stream) Close() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	stream.chart.LogStream(fmt.Sprintf("Closing stream\n"))

	stream.stopStream()
	stream.chart.Close()
	stream.buffer.Close()

	return nil
}
