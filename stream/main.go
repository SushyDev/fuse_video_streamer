package stream

import (
	"context"
	"fmt"
	"fuse_video_steamer/stream/connection"
	"fuse_video_steamer/stream/transfer"
	"sync"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

type Stream struct {
	url    string
	size   uint64
	buffer ring_buffer.LockingRingBufferInterface

	context context.Context
	cancel  context.CancelFunc

	// Job
	transfer *transfer.Transfer

	mu sync.Mutex
}

func NewStream(url string, size uint64) *Stream {
	bufferSize := min(uint64(1024*1024*1024), size)

	context, cancel := context.WithCancel(context.Background())

	buffer := ring_buffer.NewLockingRingBuffer(context, uint64(bufferSize), 0)

	manager := &Stream{
		size:   size,
		url:    url,
		buffer: buffer,

		context: context,
		cancel:  cancel,
	}

	return manager
}

func (manager *Stream) newTransferJob(startPosition uint64) error {
	connection, err := connection.NewConnection(manager.url, startPosition)
	if err != nil {
		return err
	}

	manager.buffer.ResetToPosition(startPosition)

	oldTransfer := manager.transfer
	transfer := transfer.NewTransfer(manager.buffer, connection)
	manager.transfer = transfer

	if oldTransfer != nil {
		oldTransfer.Close()
	}

	return nil
}

func (manager *Stream) ReadAt(p []byte, seekPosition uint64) (int, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.IsClosed() {
		return 0, fmt.Errorf("manager is closed")
	}

	requestedSize := uint64(len(p))
	if seekPosition+requestedSize >= manager.size {
		requestedSize = manager.size - seekPosition - 1
	}

	if !manager.buffer.IsPositionAvailable(seekPosition) {
		fmt.Println("Seek position is not available in the buffer. Starting a new connection...")

		if err := manager.newTransferJob(seekPosition); err != nil {
			return 0, err
		}
	}

	requestedPosition := min(seekPosition+requestedSize, manager.size)

	if !manager.buffer.IsPositionAvailable(requestedPosition) {
		manager.buffer.WaitForPosition(requestedPosition)
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
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.IsClosed() {
		return nil
	}

	fmt.Println("closing manager")

	if manager.transfer != nil {
		manager.transfer.Close()
	}

	manager.cancel()

	fmt.Println("closed manager")

	return nil
}
