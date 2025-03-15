package handle

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"sync"
	"syscall"
	"time"

	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	fs.Handle

	id uint64

	client    api.FileSystemServiceClient
	directory interfaces.DirectoryNode

	mu sync.RWMutex

	logger *logger.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryHandle = &Handle{}

var incrementId uint64

func New(client api.FileSystemServiceClient, directory interfaces.DirectoryNode, logger *logger.Logger) *Handle {
	incrementId++

	ctx, cancel := context.WithCancel(context.Background())

	return &Handle{
		id: incrementId,

		client:    client,
		directory: directory,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
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

	clientContext, cancel := context.WithTimeout(handle.ctx, 30*time.Second)
	defer cancel()

	response, err := handle.client.ReadDirAll(clientContext, &api.ReadDirAllRequest{
		NodeId: handle.directory.GetIdentifier(),
	})

	if err != nil {
		message := fmt.Sprintf("Failed to read directory %d", handle.directory.GetIdentifier())
		handle.logger.Error(message, err)
		return nil, err
	}

	var entries []fuse.Dirent

	for _, entry := range response.Nodes {
		switch entry.GetMode() {
		case uint32(0):
			entries = append(entries, fuse.Dirent{
				Name: entry.Name,
				Type: fuse.DT_File,
			})
		case uint32(io_fs.ModeDir):
			entries = append(entries, fuse.Dirent{
				Name: entry.Name,
				Type: fuse.DT_Dir,
			})
		}
	}

	return entries, nil
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
