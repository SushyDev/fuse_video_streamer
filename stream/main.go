package stream

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fuse_video_steamer/stream/loader"
	"fuse_video_steamer/stream/streamer/connection"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

type Stream struct {
	url  string
	size int64

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	buffer ring_buffer.LockingRingBufferInterface
	connection *connection.Connection

	mu sync.RWMutex
}

func NewStream(url string, size int64) *Stream {
	context, cancel := context.WithCancel(context.Background())

	bufferSize := min(uint64(size), 1024*1024*1024) // 1GB

	buffer := ring_buffer.NewLockingRingBuffer(context, bufferSize, 0)

	steam := &Stream{
		url:  url,
		size: size,

		context: context,
		cancel:  cancel,

		buffer: buffer,
	}

	return steam
}

func (instance *Stream) updateConnection(position int64) (*connection.Connection, error) {
	connection, err := connection.NewConnection(instance.url, position)
	if err != nil {
		fmt.Println("Failed to create new connection:", err)
		return nil, err
	}

	instance.buffer.ResetToPosition(uint64(connection.GetSeekPosition()))

	go loader.Copy(instance.buffer, connection)

	return connection, nil
}

func (instance *Stream) ReadAt(p []byte, absolutePosition int64) (int, error) {
	instance.mu.RLock()
	defer instance.mu.RUnlock()

	requestedSize := int64(len(p))

	if absolutePosition+requestedSize >= instance.size {
		absolutePosition = instance.size - absolutePosition - 1
	}

	instance.checkAndWait(p, absolutePosition)

	return instance.buffer.ReadAt(p, uint64(absolutePosition))
}

func (instance *Stream) checkAndWait(p []byte, position int64) {
	positionWithData := max(0, min(instance.size, int64(position)+int64(len(p)))) - 1

	if instance.connection == nil {
		fmt.Println("Creating new connection")
		newConnection, _ := instance.updateConnection(int64(position))
		instance.connection = newConnection

		instance.buffer.WaitForPositionInBuffer(uint64(positionWithData), 3 * time.Second)

		return 
	}

	if instance.connection.IsClosed() {
		fmt.Println("Connection is closed")
		newConnection, _ := instance.updateConnection(int64(position))
		instance.connection = newConnection

		instance.buffer.WaitForPositionInBuffer(uint64(positionWithData), 3 * time.Second)

		return
	}

	positionInBuffer := instance.buffer.IsPositionInBuffer(uint64(position))

	if !positionInBuffer {
		fmt.Println("Position is not in buffer")
		newConnection, _ := instance.updateConnection(int64(position))
		instance.connection = newConnection

		instance.buffer.WaitForPositionInBuffer(uint64(positionWithData), 3 * time.Second)

		return 
	}

	dataPositionInBuffer := instance.buffer.IsPositionInBuffer(uint64(positionWithData))

	if !dataPositionInBuffer {
		instance.buffer.WaitForPositionInBuffer(uint64(positionWithData), 3 * time.Second)
	}
}

func (stream *Stream) Close() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.connection != nil {
		stream.connection.Close()
	}

	stream.cancel()

	return nil
}
