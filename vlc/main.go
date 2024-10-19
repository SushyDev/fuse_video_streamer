package vlc

import (
	"context"
	"debrid_drive/chart"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Stream struct {
	url    string
	size   int64
	client *http.Client

	stopChannel chan struct{}

	buffer       *Buffer
	seekPosition atomic.Int64

	chart *chart.Chart

	mu sync.Mutex

	closed bool
}

var bufferCreateSize = int64(1024 * 1024 * 64)
var bufferMargin = int64(1024 * 1024 * 16)
var overflowMargin = int64(1024 * 1024 * 16)

func NewStream(url string, size int64) *Stream {
	chart := chart.NewChart()

	// buffer := NewBuffer(0, min(size, bufferCreateSize), chart)
	buffer := NewBuffer(min(size, bufferCreateSize), 0)

	client := &http.Client{
		// Transport: &http.Transport{
		// 	MaxIdleConns:        1,
		// 	MaxConnsPerHost:     1,
		// 	MaxIdleConnsPerHost: 1,
		// 	Proxy:               http.ProxyFromEnvironment,
		// },
		Timeout: time.Hour * 6,
	}

	return &Stream{
		url:    url,
		size:   size,
		client: client,

		buffer: buffer,

		chart: chart,
	}
}

func (stream *Stream) startStream(seekPosition int64, stopChannel chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		stream.chart.LogStream(fmt.Sprintf("Stream closed for position: %d\n", seekPosition))
	}()

	stream.chart.LogStream(fmt.Sprintf("Stream started for position: %d\n", seekPosition))

	rangeHeader := fmt.Sprintf("bytes=%d-", max(seekPosition, 0))
	req, err := http.NewRequestWithContext(ctx, "GET", stream.url, nil)
	if err != nil {
		stream.chart.LogStream(fmt.Sprintf("Failed to create request: %v\n", err))
		return
	}

	req.Header.Set("Range", rangeHeader)

	resp, err := stream.client.Do(req)
	if err != nil {
		stream.chart.LogStream(fmt.Sprintf("Failed to do request: %v\n", err))
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		stream.chart.LogStream(fmt.Sprintf("Status code: %d\n", resp.StatusCode))
		return
	}

	chunk := make([]byte, 1024*1024)

	timeStart := time.Now()
	var totalBytes int64

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				stream.chart.LogStream(fmt.Sprintf("Context done\n"))
				return
			case <-stopChannel:
				stream.chart.LogStream(fmt.Sprintf("Stop channel\n"))
				return
			default:
			}

			bytesToOverwrite := max(stream.buffer.GetBytesToOverwrite(), 0)
			chunkSizeToRead := min(int64(len(chunk)), bytesToOverwrite)

			// TODO chunkSizeDeterminedByNetworkSpeed

			if chunkSizeToRead == 0 {
				// timeStart = time.Now()
				continue
			}

			// fmt.Println("bytesToOverwrite", chunkSizeToRead)

			bytesRead, _ := resp.Body.Read(chunk[:chunkSizeToRead])

			if bytesRead > 0 {
				bytesWritten, err := stream.buffer.Write(chunk[:bytesRead])
				if err != nil {
					fmt.Printf("Write error %v\n", err)
				}

				totalBytes += int64(bytesWritten)
			}

			elapsed := time.Since(timeStart)
			if elapsed > 0 {
				mbps := float64(totalBytes*8) / (1024 * 1024) / elapsed.Seconds() // Convert bytes to bits and to Mbps
				stream.chart.LogStream(fmt.Sprintf("Speed: %.2f MB/s\n", mbps))
			}

			// if seekPosition+stream.buffer.Len() >= stream.size {
			// 	stream.chart.LogStream(fmt.Sprintf("Buffer is complete\n"))
			// 	continue
			// }

			// if err == io.ErrUnexpectedEOF {
			// 	stream.chart.LogStream(fmt.Sprintf("Unexpected EOF, Bytes read: %d\n", bytesRead))
			// 	return
			// }
			//
			// if err == io.EOF {
			// 	stream.chart.LogStream(fmt.Sprintf("Read EOF, Bytes read: %d\n", bytesRead))
			// 	continue
			// }
			//
			// if err != nil {
			// 	stream.chart.LogStream(fmt.Sprintf("Failed to read: %v\n", err))
			// 	continue
			// }
		}
	}()

	wg.Wait()
}

func (stream *Stream) stopStream() {
	stream.chart.LogStream(fmt.Sprintf("Stopping stream\n"))

	if stream.stopChannel != nil {
		stream.stopChannel <- struct{}{}
	}
}

func (stream *Stream) Read(p []byte) (int, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.closed {
		return 0, fmt.Errorf("Streamer is closed")
	}

	seekPosition := stream.GetSeekPosition()
	requestedSize := int64(len(p))

	if seekPosition+requestedSize >= stream.size {
		requestedSize = max(stream.size-seekPosition, 0)
	}

	stream.checkAndStartBufferIfNeeded(seekPosition, requestedSize)

	n, err := stream.buffer.ReadAt(p, seekPosition)
	if err != nil {
		fmt.Printf("ReadAt error %v\n", err)
	}

	return n, err
}

func (stream *Stream) checkAndStartBufferIfNeeded(seekPosition int64, requestedSize int64) {
	// bufferSize := stream.buffer.Cap()

	// stream.chart.LogStream(fmt.Sprintf("Check: Seek position: %d, requested size: %d\n", seekPosition, requestedSize))

	// if bufferSize >= stream.size {
	// 	stream.chart.LogStream(fmt.Sprintf("Check: Buffer is complete\n"))
	// 	return
	// }

	seekInBuffer := stream.buffer.IsPositionInBuffer(seekPosition)
	overflow := stream.buffer.OverflowByPosition(seekPosition + requestedSize)

	if !seekInBuffer || overflow >= overflowMargin {
		stream.stopStream()

		stream.buffer.Reset(seekPosition)

		// stream.chart.LogStream(fmt.Sprintf("Check: Buffer reset for position %d\n", seekPosition))

		// stream.chart.LogStream(fmt.Sprintf("Check: Starting stream for position %d\n", seekPosition))
		stopChannel := make(chan struct{})
		go stream.startStream(seekPosition, stopChannel)
		stream.stopChannel = stopChannel
		// stream.chart.LogStream(fmt.Sprintf("Check: Stream started for position %d\n", seekPosition))

		if min(seekPosition+requestedSize, stream.size) == stream.size {
		} else {
			stream.chart.LogStream(fmt.Sprintf("Check: Start: Waiting for position %d\n", seekPosition+requestedSize))
			stream.buffer.WaitForPositionInBuffer(seekPosition + requestedSize)
			stream.chart.LogStream(fmt.Sprintf("Check: Start: Position ready %d\n", seekPosition))
		}

		return
	}

	dataInBuffer := stream.buffer.IsPositionInBuffer(seekPosition + requestedSize)

	if !dataInBuffer && overflow >= 0 && overflow < overflowMargin && min(seekPosition+requestedSize, stream.size) < stream.size {
		stream.chart.LogStream(fmt.Sprintf("Check: Waiting for position %d\n", seekPosition+requestedSize))
		stream.buffer.WaitForPositionInBuffer(seekPosition + requestedSize)
		stream.chart.LogStream(fmt.Sprintf("Check: Position ready %d\n", seekPosition+requestedSize))

		return
	}

	// stream.chart.LogStream(fmt.Sprintf("Check: Position already ready %d\n", seekPosition))
}

func (stream *Stream) Seek(offset int64, whence int) (int64, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.closed {
		return 0, fmt.Errorf("Streamer is closed")
	}

	var newOffset int64

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

func (stream *Stream) GetSeekPosition() int64 {
	return stream.seekPosition.Load()
}

func (stream *Stream) GetRelativeSeekPosition() int64 {
	return stream.GetSeekPosition() - stream.buffer.GetStartPosition()
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
