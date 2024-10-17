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

var bufferCreateSize = int64(1024 * 1024 * 1024)
var bufferMargin = int64(1024 * 1024 * 128)
var overflowMargin = int64(1024 * 1024 * 16)

func NewStream(url string, size int64) *Stream {
	chart := chart.NewChart()

	buffer := NewBuffer(0, min(size, bufferCreateSize), chart)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			MaxIdleConnsPerHost: 1,
			Proxy:               http.ProxyFromEnvironment,
		},
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

	chunk := make([]byte, 1024*1024*16)
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

		currentSeekPosition := stream.GetSeekPosition()
		bytesToOverwrite := max(stream.buffer.GetBytesToOverwrite(currentSeekPosition), 0)

		chunkSizeToRead := min(1024*1024*1, bytesToOverwrite)

		// TODO chunkSizeDeterminedByNetworkSpeed

		if chunkSizeToRead == 0 {
			continue
		}

		bytesRead, err := resp.Body.Read(chunk[:chunkSizeToRead])

		if bytesRead > 0 {
			stream.buffer.Write(chunk[:bytesRead])
			continue
		}

		if seekPosition+stream.buffer.Len() >= stream.size {
			stream.chart.LogStream(fmt.Sprintf("Buffer is complete\n"))
			continue
		}

		if err == io.ErrUnexpectedEOF {
			stream.chart.LogStream(fmt.Sprintf("Unexpected EOF, Bytes read: %d\n", bytesRead))
			return
		}

		if err == io.EOF {
			stream.chart.LogStream(fmt.Sprintf("Read EOF, Bytes read: %d\n", bytesRead))
			continue
		}

		if err != nil {
			stream.chart.LogStream(fmt.Sprintf("Failed to read: %v\n", err))
			continue
		}
	}
}

func (stream *Stream) stopStream() {
	stream.chart.LogStream(fmt.Sprintf("Check: Stopping buffer for position\n"))

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

	return stream.buffer.ReadAt(p, seekPosition)
}

func (stream *Stream) checkAndStartBufferIfNeeded(seekPosition int64, requestedSize int64) {
	bufferSize := stream.buffer.Len()

	stream.chart.LogStream(fmt.Sprintf("Check: Seek position: %d, requested size: %d, buffer size: %d\n", seekPosition, requestedSize, bufferSize))

	if bufferSize >= stream.size {
		stream.chart.LogStream(fmt.Sprintf("Check: Buffer is complete\n"))
		return
	}

	seekInBuffer := stream.buffer.IsPositionInBuffer(seekPosition)
	overflow := stream.buffer.OverflowByPosition(seekPosition + requestedSize)

	if !seekInBuffer || overflow >= overflowMargin {
		stream.stopStream()

		stream.buffer.Reset()
		stream.buffer.SetStartPosition(seekPosition)

		stream.chart.LogStream(fmt.Sprintf("Check: Buffer reset for position %d\n", seekPosition))

		stopChannel := make(chan struct{})
		go stream.startStream(seekPosition, stopChannel)

		stream.buffer.WaitForPositionInBuffer(seekPosition + requestedSize)
		stream.stopChannel = stopChannel

		stream.chart.LogStream(fmt.Sprintf("Check: Buffer started for position %d\n", seekPosition))

		return
	}

	dataInBuffer := stream.buffer.IsPositionInBuffer(seekPosition + requestedSize)

	if !dataInBuffer && overflow >= 0 && overflow-bufferMargin < overflowMargin {
		stream.chart.LogStream(fmt.Sprintf("Check: Waiting for position %d\n", seekPosition+requestedSize))
		stream.buffer.WaitForPositionInBuffer(seekPosition + requestedSize)
		stream.chart.LogStream(fmt.Sprintf("Position %d is ready\n", seekPosition+requestedSize))
	} else {
		stream.chart.LogStream(fmt.Sprintf("Check: Buffer is ready for position %d\n", seekPosition+requestedSize))
	}
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
