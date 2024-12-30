package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream/buffer"
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

	logger *logger.Logger

	mu sync.RWMutex

	closed bool
}

var bufferCreateSize = uint64(1024 * 1024 * 1024 * 1)

func NewStream(url string, size uint64) *Stream {
	logger, err := logger.NewLogger("Stream")
	if err != nil {
		panic(err)
	}

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

		logger: logger,
	}
}

func (stream *Stream) startStream(seekPosition uint64) {
	defer func() {
		stream.wg.Done()
	}()

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	rangeHeader := fmt.Sprintf("bytes=%d-", max(seekPosition, 0))
	req, err := http.NewRequestWithContext(ctx, "GET", stream.url, nil)
	if err != nil {
		message := fmt.Sprintf("Stream \"%s\" failed to create request", stream.url)
		stream.logger.Error(message, err)
		stream.cancel()
		return
	}

	req.Header.Set("Range", rangeHeader)

	resp, err := stream.client.Do(req)
	if err != nil {
		message := fmt.Sprintf("Stream \"%s\" failed to do request", stream.url)
		stream.logger.Error(message, err)
		stream.cancel()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		stream.logger.Error(fmt.Sprintf("Stream \"%s\" failed to get partial content: %d", stream.url, resp.StatusCode), nil)
		stream.cancel()
		return
	}

	chunk := make([]byte, 8192)

	var normalDelay time.Duration = 100 * time.Microsecond
	var retryDelay time.Duration = normalDelay

	for {
		select {
		case <-stream.context.Done():
			return
		case <-ctx.Done():
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
			_, err := stream.buffer.Write(chunk[:bytesRead])
			if err != nil {
				message := fmt.Sprintf("Stream \"%s\" failed to write", stream.url)
				stream.logger.Error(message, err)
				return // Crash ?
			}

			retryDelay = normalDelay // Reset
		}

		switch {
		case err == io.ErrUnexpectedEOF:
			message := fmt.Sprintf("Stream \"%s\" unexpected EOF, Bytes read: %d", stream.url, bytesRead)
			stream.logger.Error(message, nil)
			return // Decide if the loop should crash or retry logic can be added.
		case err == io.EOF:
			// TODO FINISHED
			retryDelay = 5 * time.Second
			continue
		case err != nil:
			message := fmt.Sprintf("Stream \"%s\" failed to read", stream.url)
			stream.logger.Error(message, err)
			retryDelay = 5 * time.Second
			continue
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
	stream.mu.RLock()
	defer stream.mu.RUnlock()

	if stream.closed {
		return 0, fmt.Errorf("Streamer is closed")
	}

	seekPosition := stream.GetSeekPosition()
	requestedSize := uint64(len(p))

	if seekPosition+requestedSize >= stream.size {
		requestedSize = stream.size - seekPosition - 1
	}

	stream.checkAndStartBufferIfNeeded(seekPosition, requestedSize)

	n, err := stream.buffer.ReadAt(p, seekPosition)
	if err != nil {
		return 0, fmt.Errorf("ReadAt error %v\n", err)
	}

	return n, err
}

func (stream *Stream) checkAndStartBufferIfNeeded(seekPosition uint64, requestedSize uint64) {
	if seekPosition >= stream.size {
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

		stream.buffer.WaitForPositionInBuffer(stream.context, waitForSize)

		return
	}

	dataInBuffer := stream.buffer.IsPositionInBufferSync(seekPosition + requestedSize)

	if !dataInBuffer {
		waitForSize := min(seekPosition+requestedSize, stream.size)

		stream.buffer.WaitForPositionInBuffer(stream.context, waitForSize)
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
		return 0, fmt.Errorf("SeekCurrent is not supported")
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

	return newOffset, err
}

func (stream *Stream) GetSeekPosition() uint64 {
	return stream.seekPosition.Load()
}

func (stream *Stream) Close() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	stream.stopStream()
	stream.buffer.Close()

	return nil
}
