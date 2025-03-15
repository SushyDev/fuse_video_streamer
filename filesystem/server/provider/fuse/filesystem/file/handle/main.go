package handle

import (
	"context"
	"sync"
	"syscall"

	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	node interfaces.FileNode

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	id uint64

	logger *logger.Logger

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileHandle = &Handle{}

var incrementId uint64

func New(node interfaces.FileNode, logger *logger.Logger) *Handle {
	incrementId++

	ctx, cancel := context.WithCancel(context.Background())

	return &Handle{
		node: node,

		id: incrementId,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.isClosed() {
		return nil, syscall.ENOENT
	}

	client := handle.node.GetClient()

	response, err := client.ReadFile(ctx, &api.ReadFileRequest{
		NodeId: handle.node.GetIdentifier(),
		Offset: 0,
		Size:   handle.node.GetSize(),
	})

	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.isClosed() {
		return syscall.ENOENT
	}

	client := handle.node.GetClient()

	response, err := client.ReadFile(ctx, &api.ReadFileRequest{
		NodeId: handle.node.GetIdentifier(),
		Offset: uint64(readRequest.Offset),
		Size:   uint64(readRequest.Size),
	})

	if err != nil {
		return err
	}

	readResponse.Data = response.Data

	return nil
}

func (handle *Handle) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.isClosed() {
		return syscall.ENOENT
	}

	client := handle.node.GetClient()

	response, err := client.WriteFile(ctx, &api.WriteFileRequest{
		NodeId: handle.node.GetIdentifier(),
		Offset: uint64(writeRequest.Offset),
		Data:   writeRequest.Data,
	})

	if err != nil {
		return err
	}

	writeResponse.Size = int(response.BytesWritten)

	return nil
}

func (handle *Handle) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.isClosed() {
		return syscall.ENOENT
	}

	return nil
}

func (handle *Handle) Close() error {
	// handle.mu.Lock()
	// defer handle.mu.Unlock()

	if handle.isClosed() {
		return nil
	}

	handle.cancel()

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
