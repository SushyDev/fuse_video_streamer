package directory

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"github.com/anacrolix/fuse"
)

type Handle struct {
	id uint64

	directory interfaces_node.DirectoryNode

	mu sync.RWMutex

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_handle.DirectoryHandle = &Handle{}

var incrementId uint64

func NewHandle(directory interfaces_node.DirectoryNode, logger interfaces_logger.Logger) *Handle {
	incrementId++

	handle := &Handle{
		id: incrementId,

		directory: directory,

		logger: logger,
	}

	return handle
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	if handle.IsClosed() {
		return nil, syscall.ENOENT
	}

	handle.mu.RLock()
	defer handle.mu.RUnlock()

	fileSystem := handle.directory.GetClient().GetFileSystem()

	nodes, err := fileSystem.ReadDirAll(handle.directory.GetRemoteIdentifier())
	if err != nil && err != syscall.ENOENT {
		message := fmt.Sprintf("failed to read directory %d", handle.directory.GetRemoteIdentifier())
		handle.logger.Error(message, err)
		return nil, err
	}

	var entries []fuse.Dirent

	for _, entry := range nodes {
		mode := entry.GetMode()

		// Check file type using mode bits, not exact equality
		if mode.IsDir() {
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_Dir,
			})
		} else if mode.IsRegular() || mode == 0 {
			// Regular files, including hardlinks
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_File,
			})
		} else {
			message := fmt.Sprintf("unknown file mode %d (0x%x) for file %s", mode, mode, entry.GetName())
			handle.logger.Error(message, nil)
		}
	}

	return entries, nil
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
