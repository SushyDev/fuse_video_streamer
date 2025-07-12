package handle

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"syscall"

	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/pool"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/stream"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	node interfaces.StreamableNode

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	id uint64

	stream *stream.Stream

	logger *logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces.StreamableHandle = &Handle{}

var incrementId uint64

func New(node interfaces.StreamableNode, stream *stream.Stream, logger *logger.Logger) *Handle {
	incrementId++

	return &Handle{
		node: node,

		id: incrementId,

		stream: stream,

		logger: logger,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	if handle.stream == nil {
		message := fmt.Sprintf("No video stream for handle %d, closing video stream", handle.id)
		handle.logger.Error(message, nil)

		handle.Close()

		return syscall.ENOENT
	}

	fileSize := handle.node.GetSize()

	// Use requested size instead of file size for buffer allocation
	requestedSize := int64(readRequest.Size)
	bufferSize := min(requestedSize, fileSize)
	
	buffer := pool.GetBuffer(bufferSize)
	defer pool.PutBuffer(buffer)

	// Only allocate what we need from the buffer
	readBuffer := buffer[:readRequest.Size]
	bytesRead, err := handle.stream.ReadAt(readBuffer, readRequest.Offset)

	switch err {

	case nil:
		readResponse.Data = readBuffer[:bytesRead]
		return nil

	case io.EOF:
		readResponse.Data = readBuffer[:bytesRead]
		return nil

	default:
		message := fmt.Sprintf("Failed to read video stream for handle %d, closing video stream", handle.id)
		handle.logger.Error(message, err)

		handle.stream.Close()
		handle.stream = nil

		return err
	}
}

func (handle *Handle) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	handle.Close()

	return nil
}

func (handle *Handle) Close() error {
	if !handle.closed.CompareAndSwap(false, true) {
		return nil
	}

	if handle.stream != nil {
		handle.stream.Close()
	}

	return nil
}

func (handle *Handle) IsClosed() bool {
	return handle.closed.Load()
}
