package stream

import (
	"context"
	"fmt"
	"fuse_video_steamer/stream/connection"
	"fuse_video_steamer/stream/transfer"
	"sync"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

const (
	maxBufferSize  = int64(1024 * 1024 * 1024) // 1GB
	maxPreloadSize = int64(16 * 1024 * 1024) // 16MB
)

type Stream struct {
	pid  uint32
	id  string
	url  string
	size int64

	buffer ring_buffer.LockingRingBufferInterface

	context context.Context
	cancel  context.CancelFunc

	// Job
	transfer *transfer.Transfer

	mu sync.Mutex
}

// Buffer size is 10% of buffer size, capped at 1GB or fileSize
func calculateBufferSize(fileSize int64) int64 {
	bufferSize := int64(float64(fileSize) * 0.1)
	return min(maxBufferSize, bufferSize, fileSize)
}

// Preload size is half the buffer size, capped at 16 MB or buffer size
func calculatePreloadSize(bufferSize int64) int64 {
	preloadSize := int64(float64(bufferSize) * 0.5)
	return min(maxPreloadSize, preloadSize, bufferSize)
}

func NewStream(pid uint32, url string, size int64) *Stream {
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	bufferSize := calculateBufferSize(int64(size))

	buffer := ring_buffer.NewLockingRingBuffer(bufferSize, 0)

	context, cancel := context.WithCancel(context.Background())

	stream := &Stream{
		pid: pid,
		id: id,

		size: size,
		url:  url,

		buffer: buffer,

		context: context,
		cancel:  cancel,
	}

	return stream
}

func (stream *Stream) ReadAt(p []byte, seekPosition int64) (int, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.IsClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	requestedBytes := int64(len(p))

	if !stream.buffer.IsPositionAvailable(seekPosition) {
		if err := stream.newTransfer(seekPosition); err != nil {
			return 0, err
		}
	}

	requestedPosition := min(seekPosition+requestedBytes, stream.size)

	if !stream.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(stream.context, 30*time.Second)
		defer cancel()

		ok := stream.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok {
			return 0, fmt.Errorf("Timeout waiting for the buffer to fill")
		}
	}

	return stream.buffer.ReadAt(p, seekPosition)
}

func (stream *Stream) Close() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.IsClosed() {
		return nil
	}

	stream.cancel()

	if stream.buffer != nil {
		err := stream.buffer.Close()
		if err != nil {
			return fmt.Errorf("Error closing buffer: %v", err)
		}

		stream.buffer = nil
	}

	if stream.transfer != nil {
		err := stream.transfer.Close()
		if err != nil {
			fmt.Println("Error closing transfer:", err)
		}

		stream.transfer = nil
	}

	return nil
}

func (stream *Stream) IsClosed() bool {
	select {
	case <-stream.context.Done():
		return true
	default:
		return false
	}
}

func (stream *Stream) newTransfer(startPosition int64) error {
	if stream.transfer != nil {
		stream.transfer.Close()
		stream.transfer = nil
	}

	bufferSize := calculateBufferSize(stream.size)
	preloadSize := calculatePreloadSize(bufferSize)

	streamStartPosition := max(0, startPosition-preloadSize)

	connection, err := connection.NewConnection(stream.url, streamStartPosition)
	if err != nil {
		return err
	}

	stream.buffer.ResetToPosition(streamStartPosition)
	transfer := transfer.NewTransfer(stream.buffer, connection)
	stream.transfer = transfer

	return nil
}
