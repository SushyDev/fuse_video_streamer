package handle

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"sync"
	"sync/atomic"
	"syscall"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	fs.Handle

	id uint64

	client    filesystem_client_interfaces.Client
	directory interfaces.DirectoryNode

	mu sync.RWMutex

	logger *logger.Logger

	closed atomic.Bool
}

var _ interfaces.DirectoryHandle = &Handle{}

var incrementId uint64

func New(client filesystem_client_interfaces.Client, directory interfaces.DirectoryNode, logger *logger.Logger) *Handle {
	incrementId++

	return &Handle{
		id: incrementId,

		client:    client,
		directory: directory,

		logger: logger,
	}
}

func (handle *Handle) GetIdentifier() uint64 {
	return handle.id
}

func (handle *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.isClosed() {
		return nil, syscall.ENOENT
	}

	fileSystem := handle.client.GetFileSystem()

	nodes, err := fileSystem.ReadDirAll(handle.directory.GetIdentifier())
	if err != nil && err != syscall.ENOENT {
		message := fmt.Sprintf("Failed to read directory %d", handle.directory.GetIdentifier())
		handle.logger.Error(message, err)
		return nil, err
	}

	var entries []fuse.Dirent

	for _, entry := range nodes {
		switch entry.GetMode() {
		case io_fs.ModeSymlink:
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_Link,
			})
		case io_fs.FileMode(0):
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_File,
			})
		case io_fs.ModeDir:
			entries = append(entries, fuse.Dirent{
				Name: entry.GetName(),
				Type: fuse.DT_Dir,
			})
		default:
			message := fmt.Sprintf("Unknown file mode %s", entry.GetName())
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

func (handle *Handle) isClosed() bool {
	return handle.closed.Load()
}
