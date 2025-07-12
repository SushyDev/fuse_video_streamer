package handle

import (
	"context"
	"sync"
	"sync/atomic"
	"syscall"

	"fuse_video_streamer/filesystem/driver/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

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

	closed atomic.Bool
}

var _ interfaces.FileHandle = &Handle{}

var incrementId uint64

func New(node interfaces.FileNode, logger *logger.Logger) *Handle {
	incrementId++

	return &Handle{
		node: node,

		id: incrementId,

		logger: logger,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.IsClosed() {
		return nil, syscall.ENOENT
	}

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	data, err := fileSystem.ReadFile(handle.node.GetIdentifier(), 0, handle.node.GetSize())

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	data, err := fileSystem.ReadFile(handle.node.GetIdentifier(), uint64(readRequest.Offset), uint64(readRequest.Size))
	if err != nil {
		return err
	}

	readResponse.Data = data

	return nil
}

func (handle *Handle) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	bytesWritten, err := fileSystem.WriteFile(handle.node.GetIdentifier(), uint64(writeRequest.Offset), writeRequest.Data)
	if err != nil {
		return err
	}

	writeResponse.Size = int(bytesWritten)

	return nil
}

func (handle *Handle) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	return nil
}

func (handle *Handle) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	return nil
}

func (handle *Handle) Fsync(ctx context.Context, fsyncRequest *fuse.FsyncRequest) error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.IsClosed() {
		return syscall.ENOENT
	}

	return nil
}

func (handle *Handle) Close() error {
	if !handle.closed.CompareAndSwap(false, true) {
		return nil
	}

	return nil
}

func (handle *Handle) IsClosed() bool {
	return handle.closed.Load()
}
