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
	minBufferSize  = int64(100 * 1024 * 1024)  // 100MB
	maxPreloadSize = int64(25 * 1024 * 1024) // 25MB
	minPreloadSize = int64(10 * 1024 * 1024)   // 10MB
)

type Stream struct {
	url  string
	size int64

	buffer ring_buffer.LockingRingBufferInterface

	context context.Context
	cancel  context.CancelFunc

	// Job
	transfer *transfer.Transfer

	mu sync.Mutex
}

// Buffer size is 10% of buffer size, capped at 1GB or at least 100 mb unless file size is less than 100 mb then its the file size
func calculateBufferSize(fileSize int64) int64 {
	bufferSize := int64(float64(fileSize) * 0.1)
	return min(maxBufferSize, bufferSize, fileSize)
}

// Preload size is half the buffer size, capped at 200 MB or at least 10 mb unless the buffer size is less than 10 mb then its the buffer size
func calculatePreloadSize(bufferSize int64) int64 {
	preloadSize := int64(float64(bufferSize) * 0.5)
	return min(maxPreloadSize, preloadSize, bufferSize)
}

func NewStream(url string, size int64) *Stream {
	bufferSize := calculateBufferSize(int64(size))

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

func (manager *Stream) ReadAt(p []byte, seekPosition int64) (int, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.IsClosed() {
		return 0, fmt.Errorf("manager is closed")
	}

	requestedBytes := int64(len(p))

	if !manager.buffer.IsPositionAvailable(seekPosition) {
		if err := manager.newTransfer(seekPosition); err != nil {
			return 0, err
		}

		return manager.buffer.ReadAt(p, seekPosition)
	}

	requestedPosition := min(seekPosition+requestedBytes, manager.size)

	if !manager.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(manager.context, 100*time.Second)
		defer cancel()

		ok := manager.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok {
			return 0, fmt.Errorf("Timeout waiting for the buffer to fill")
		}
	}

	return manager.buffer.ReadAt(p, seekPosition)
}

func (manager *Stream) Close() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.IsClosed() {
		return nil
	}

	if manager.transfer != nil {
		err := manager.transfer.Close()
		if err != nil {
			fmt.Println("Error closing transfer:", err)
		}

		manager.transfer = nil
	}

	if manager.buffer != nil {
		err := manager.buffer.Close()
		if err != nil {
			return fmt.Errorf("Error closing buffer: %v", err)
		}

		manager.buffer = nil
	}

	manager.cancel()

	return nil
}

func (manager *Stream) IsClosed() bool {
	select {
	case <-manager.context.Done():
		return true
	default:
		return false
	}
}

func (manager *Stream) newTransfer(startPosition int64) error {
	if manager.transfer != nil {
		manager.transfer.Close()
		manager.transfer = nil
	}

	bufferSize := calculateBufferSize(manager.size)
	preloadSize := calculatePreloadSize(bufferSize)

	streamStartPosition := max(0, startPosition-preloadSize)

	connection, err := connection.NewConnection(manager.url, streamStartPosition)
	if err != nil {
		return err
	}

	manager.buffer.ResetToPosition(streamStartPosition)
	transfer := transfer.NewTransfer(manager.buffer, connection)
	manager.transfer = transfer

	ctx, cancel := context.WithTimeout(manager.context, 100*time.Second)
	defer cancel()

	streamWaitPosition := startPosition+preloadSize

	if streamWaitPosition > manager.size {
		streamWaitPosition = manager.size
	}

	ok := manager.buffer.WaitForPosition(ctx, streamWaitPosition)
	if !ok {
		return fmt.Errorf("Timeout waiting for the buffer to fill while starting a new transfer")
	}

	return nil
}
