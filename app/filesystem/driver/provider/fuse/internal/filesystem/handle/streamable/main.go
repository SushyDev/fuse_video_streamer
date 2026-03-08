package streamable

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_logger "fuse_video_streamer/logger/interfaces"
	interfaces_stream "fuse_video_streamer/stream/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	node interfaces_node.StreamableNode

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	id uint64

	stream interfaces_stream.Stream

	bufferPool pool.BufferPool

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_handle.StreamableHandle = &Handle{}

var incrementId uint64

func NewHandle(node interfaces_node.StreamableNode, stream interfaces_stream.Stream, bufferPool pool.BufferPool, logger interfaces_logger.Logger) *Handle {
	id := atomic.AddUint64(&incrementId, 1)

	return &Handle{
		node: node,

		id: id,

		stream: stream,

		bufferPool: bufferPool,

		logger: logger,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	if handle.IsClosed() {
		message := fmt.Sprintf("handle %d is closed, cannot read from video stream", handle.id)
		handle.logger.Error(message, nil)
		return syscall.ENOENT
	}

	// Use RLock to allow concurrent FUSE reads. The stream's internal
	// synchronization handles concurrent ReadAt calls safely.
	handle.mu.RLock()
	stream := handle.stream
	handle.mu.RUnlock()

	if stream == nil {
		message := fmt.Sprintf("no video stream for handle %d", handle.id)
		handle.logger.Error(message, nil)
		handle.Close()
		return syscall.ENOENT
	}

	if stream.IsClosed() {
		message := fmt.Sprintf("video stream for handle %d is closed, cannot read from video stream", handle.id)
		handle.logger.Error(message, nil)
		handle.Close()
		return syscall.ENOENT
	}

	fileSize := handle.node.GetSize()
	_ = fileSize

	buffer := handle.bufferPool.Get(readRequest.Size)
	defer handle.bufferPool.Put(buffer)

	// Pass the FUSE request context so cancellation propagates all the way
	// down to WaitForPosition and network reads.
	bytesRead, err := stream.ReadAt(ctx, buffer[:readRequest.Size], readRequest.Offset)

	switch err {

	case nil:
		readResponse.Data = buffer[:bytesRead]
		return nil

	case io.EOF:
		readResponse.Data = buffer[:bytesRead]
		return nil

	case context.Canceled, context.DeadlineExceeded:
		// FUSE client abandoned the read; don't kill the stream.
		return syscall.EINTR

	default:
		message := fmt.Sprintf("failed to read video stream for handle %d, closing video stream", handle.id)
		handle.logger.Error(message, err)

		stream.Close()

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
