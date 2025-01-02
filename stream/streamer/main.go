package streamer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"fuse_video_steamer/stream/streamer/connection"
)

var _ io.Seeker = &Stream{}
var _ io.Closer = &Stream{}

type Stream struct {
	url    string
	size   int64

	stopChannel chan struct{}
	waitChannel chan struct{}

	context context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	seekPosition atomic.Int64

	connection *connection.Connection

	mu sync.RWMutex

	closed bool // todo change for context
}

func NewStream(url string, size int64) *Stream {
	steam := &Stream{
		url:    url,
		size:   size,
	}

	return steam
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
		return stream.seekPosition.Load(), nil
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
		newOffset = stream.size - 1
		err = io.EOF
	}

	stream.seekPosition.Store(newOffset)

	return newOffset, err
}

func (stream *Stream) GetConnection() *connection.Connection {
	return stream.connection
}

func (stream *Stream) NewConnection() (*connection.Connection, error) {
	seekPosition := stream.seekPosition.Load()

	connection, err := connection.NewConnection(stream.url, seekPosition)
	if err != nil {
		return nil, fmt.Errorf("Failed to create connection: %v", err)
	}

	if stream.connection != nil {
		stream.connection.Close()
	}

	stream.connection = connection

	return connection, nil
}

func (stream *Stream) Close() error {
	if stream.closed {
		return fmt.Errorf("Streamer is already closed")
	}

	if stream.connection != nil {
		stream.connection.Close()
	}

	stream.cancel()

	return nil
}
