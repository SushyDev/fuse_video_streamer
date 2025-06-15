package handle

import (
	"context"
	"fmt"
	"io"
	"sync"
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

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.StreamableHandle = &Handle{}

var incrementId uint64

func New(node interfaces.StreamableNode, stream *stream.Stream, logger *logger.Logger) *Handle {
	incrementId++

	ctx, cancel := context.WithCancel(context.Background())

	return &Handle{
		node: node,

		id: incrementId,

		stream: stream,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.isClosed() {
		return syscall.ENOENT
	}

	if handle.stream == nil {
		message := fmt.Sprintf("No video stream for handle %d, closing video stream", handle.id)
		handle.logger.Error(message, nil)

		handle.Close()

		return syscall.ENOENT
	}

	fileSize := handle.node.GetSize()

	buffer := pool.GetBuffer(int64(fileSize))
	defer pool.PutBuffer(buffer)

	bytesRead, err := handle.stream.ReadAt(buffer[:readRequest.Size], readRequest.Offset)
	switch err {
	case nil:
		readResponse.Data = buffer[:bytesRead]
		return nil

	case io.EOF:
		readResponse.Data = buffer[:bytesRead]
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
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.isClosed() {
		return nil
	}

	handle.cancel()

	if handle.stream != nil {
		handle.stream.Close()
		handle.stream = nil
	}

	return nil
}

func (handle *Handle) isClosed() bool {
	select {
	case <-handle.ctx.Done():
		return true
	default:
		return false
	}
}
