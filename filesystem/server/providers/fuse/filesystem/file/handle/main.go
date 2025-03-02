package handle

import (
	"context"
	"fmt"
	"io"
	"sync"
	"syscall"

	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/providers/fuse/pool"
	"fuse_video_steamer/cache"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	id  uint64

	cache *cache.Cache

	logger *logger.Logger

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileHandle = &Handle{}

var incrementId uint64

func New(cache *cache.Cache, logger *logger.Logger) *Handle {
	incrementId++

	ctx, cancel := context.WithCancel(context.Background())

	return &Handle{
		id: incrementId,

		cache: cache,

		logger: logger,

		ctx: ctx,
		cancel: cancel,
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

	buffer := pool.GetBuffer()
	defer pool.PutBuffer(buffer)

	bytesRead, err := handle.cache.ReadAt(buffer[:readRequest.Size], readRequest.Offset)
	switch err {
	case nil:
		readResponse.Data = buffer[:bytesRead]
		return nil

	case io.EOF:
		readResponse.Data = buffer[:bytesRead]
		return nil

	default:
		message := fmt.Sprintf("Failed to read video stream for handl %d, closing video stream", handle.id)
		handle.logger.Error(message, err)

		handle.cache.Close()

		return err
	}
}

func (handle *Handle) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	handle.cache.Close()
	handle.cache = nil

	return nil
}


func (handle *Handle) Close() error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.IsClosed() {
		return nil
	}

	handle.cancel()

	handle.cache.Close()
	handle.cache = nil

	return nil
}

func (handle *Handle) IsClosed() bool {
	select {
	case <-handle.ctx.Done():
		return true
	default:
		return false
	}
}
