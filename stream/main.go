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

var standardBufferSize uint64 = 1024 * 1024 * 1024

type Stream struct {
	url  string
	size uint64

	buffer ring_buffer.LockingRingBufferInterface

	context context.Context
	cancel  context.CancelFunc

	// Job
	transfer *transfer.Transfer

	mu sync.Mutex
}

func calculateBufferSize(fileSize int64) int64 {
	maxBufferSize := int64(1024 * 1024 * 1024)
	minBufferSize := max(100*1024*1024, fileSize)

	// Buffer size is 10% of buffer size, capped at 1GB or at least 100 mb or max buffer size
	return min(int64(min(float64(fileSize)*0.1, float64(maxBufferSize))), minBufferSize)
}

func calculatePreloadSize(bufferSize int64) int64 {
	maxPreloadSize := float64(1024 * 1024 * 25)
	minPreloadSize := max(10*1024*1024, bufferSize)

	// Preload size is half the buffer size, capped at 200 MB or at least 10 mb or max buffer size
	return min(int64(min(float64(bufferSize)*0.5, maxPreloadSize)), minPreloadSize)
}

func NewStream(url string, size uint64) *Stream {
	bufferSize := uint64(calculateBufferSize(int64(size)))

	buffer := ring_buffer.NewLockingRingBuffer(bufferSize, 0)

	context, cancel := context.WithCancel(context.Background())

	manager := &Stream{
		size: size,
		url:  url,

		buffer: buffer,

		context: context,
		cancel:  cancel,
	}

	return manager
}

func (manager *Stream) newTransfer(startPosition uint64) error {
	if manager.transfer != nil {
		manager.transfer.Close()
	}

	preloadSize := calculatePreloadSize(int64(manager.buffer.Size()))

	bufferStartPosition := uint64(max(0, int64(startPosition)-preloadSize))

	connection, err := connection.NewConnection(manager.url, bufferStartPosition)
	if err != nil {
		return err
	}

	manager.buffer.ResetToPosition(bufferStartPosition)
	transfer := transfer.NewTransfer(manager.buffer, connection)
	manager.transfer = transfer

	ctx, cancel := context.WithTimeout(manager.context, 10*time.Second)
	defer cancel()

	ok := manager.buffer.WaitForPosition(ctx, min(manager.size, bufferStartPosition+uint64(preloadSize)))
	if !ok {
		return fmt.Errorf("Timeout waiting for the buffer to fill")
	}

	return nil
}

func (manager *Stream) ReadAt(p []byte, seekPosition uint64) (int, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.IsClosed() {
		return 0, fmt.Errorf("manager is closed")
	}

	if !manager.buffer.IsPositionAvailable(seekPosition) {
		fmt.Println("Seek position is not available in the buffer. Starting a new connection...")

		if err := manager.newTransfer(seekPosition); err != nil {
			return 0, err
		}

		return manager.buffer.ReadAt(p, seekPosition)
	}

	requestedSize := uint64(len(p))
	requestedPosition := min(seekPosition+requestedSize, manager.size)

	if !manager.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(manager.context, 10*time.Second)
		defer cancel()

		ok := manager.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok {
			return 0, fmt.Errorf("Timeout waiting for the buffer to fill")
		}
	}

	return manager.buffer.ReadAt(p, uint64(seekPosition))
}

func (manager *Stream) IsClosed() bool {
	select {
	case <-manager.context.Done():
		return true
	default:
		return false
	}
}

func (manager *Stream) Close() error {
	if manager.IsClosed() {
		return nil
	}

	if manager.transfer != nil {
		manager.transfer.Close()
	}

	if manager.buffer != nil {
		manager.buffer.Close()
	}

	manager.cancel()

	return nil
}
