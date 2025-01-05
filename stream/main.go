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
	size uint64

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
	return min(maxBufferSize, min(bufferSize, fileSize))
}

// Preload size is half the buffer size, capped at 200 MB or at least 10 mb unless the buffer size is less than 10 mb then its the buffer size
func calculatePreloadSize(bufferSize int64) int64 {
	preloadSize := int64(float64(bufferSize) * 0.5)
	return min(maxPreloadSize, min(preloadSize, bufferSize))
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

	streamStartPosition := uint64(max(0, int64(startPosition)-preloadSize))

	connection, err := connection.NewConnection(manager.url, streamStartPosition)
	if err != nil {
		return err
	}

	manager.buffer.ResetToPosition(streamStartPosition)
	transfer := transfer.NewTransfer(manager.buffer, connection)
	manager.transfer = transfer

	ctx, cancel := context.WithTimeout(manager.context, 10*time.Second)
	defer cancel()

	streamWaitPosition := uint64(int64(startPosition)+preloadSize)

	if streamWaitPosition > manager.size {
		fmt.Println("Stream wait position is greater than the file size. Setting it to the end of the file.")
		streamWaitPosition = manager.size - 1
	}

	ok := manager.buffer.WaitForPosition(ctx, streamWaitPosition)
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

	requestedBytes := uint64(len(p))

	if !manager.buffer.IsPositionAvailable(seekPosition) {
		fmt.Println("Seek position is not available in the buffer. Starting a new connection...")

		if err := manager.newTransfer(seekPosition); err != nil {
			return 0, err
		}
	}

	requestedPosition := min(seekPosition+requestedBytes, manager.size)

	if !manager.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(manager.context, 5*time.Second)
		defer cancel()

		ok := manager.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok {
			return 0, fmt.Errorf("Timeout waiting for the buffer to fill")
		}
	}

	return manager.buffer.ReadAt(p, seekPosition)
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
	manager.mu.Lock()
	defer manager.mu.Unlock()

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
