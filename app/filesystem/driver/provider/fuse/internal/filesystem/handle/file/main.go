package file

import (
	"context"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"github.com/anacrolix/fuse"
)

type Handle struct {
	id   uint64
	node interfaces_node.FileNode

	logger interfaces_logger.Logger

	mu     sync.RWMutex
	closed atomic.Bool
}

var _ interfaces_handle.FileHandle = &Handle{}

var incrementId uint64

func NewHandle(node interfaces_node.FileNode, logger interfaces_logger.Logger) *Handle {
	incrementId++

	return &Handle{
		id:   incrementId,
		node: node,

		logger: logger,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) ReadAll(ctx context.Context) ([]byte, error) {
	if handle.IsClosed() {
		return nil, syscall.ENOENT
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	data, err := fileSystem.ReadFile(handle.node.GetRemoteIdentifier(), 0, handle.node.GetSize())

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (handle *Handle) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	if handle.IsClosed() {
		return syscall.ENOENT
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	data, err := fileSystem.ReadFile(handle.node.GetRemoteIdentifier(), uint64(readRequest.Offset), uint64(readRequest.Size))
	if err != nil {
		return err
	}

	readResponse.Data = data

	return nil
}

func (handle *Handle) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	if handle.IsClosed() {
		return syscall.ENOENT
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	client := handle.node.GetClient()
	fileSystem := client.GetFileSystem()

	bytesWritten, err := fileSystem.WriteFile(handle.node.GetRemoteIdentifier(), uint64(writeRequest.Offset), writeRequest.Data)
	if err != nil {
		return err
	}

	writeResponse.Size = int(bytesWritten)

	// Refresh the file size after write
	newSize, _, err := fileSystem.GetFileInfo(handle.node.GetRemoteIdentifier())
	if err == nil {
		handle.node.UpdateSize(newSize)
	}

	return nil
}

func (handle *Handle) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	if handle.IsClosed() {
		return syscall.ENOENT
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	return nil
}

func (handle *Handle) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	if handle.IsClosed() {
		return syscall.ENOENT
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

	return nil
}

func (handle *Handle) Fsync(ctx context.Context, fsyncRequest *fuse.FsyncRequest) error {
	if handle.IsClosed() {
		return syscall.ENOENT
	}

	handle.mu.Lock()
	defer handle.mu.Unlock()

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
