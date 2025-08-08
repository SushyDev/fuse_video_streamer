package directory

import (
	"context"
	"fmt"
	io_fs "io/fs"
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
		switch entry.GetMode() {
		// --- Symlink
		case io_fs.ModeSymlink:
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_Link,
			})
		// --- File
		case io_fs.FileMode(0):
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_File,
			})
		// --- Directory
		case io_fs.ModeDir:
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_Dir,
			})
		// --- Unknown
		default:
			message := fmt.Sprintf("unknown file mode %s for file %s", entry.GetMode(), entry.GetName())
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
