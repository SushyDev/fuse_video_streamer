package handle

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"

	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Handle struct {
	fs.Handle

	client vfs_api.FileSystemServiceClient
	directory interfaces.DirectoryNode

	mu sync.RWMutex

	logger *logger.Logger

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryHandle = &Handle{}

func New(client vfs_api.FileSystemServiceClient, directory interfaces.DirectoryNode, logger *logger.Logger) *Handle {
	ctx, cancel := context.WithCancel(context.Background())

	return &Handle{
		client: client,
		directory: directory,

		logger: logger,

		ctx: ctx,
		cancel: cancel,
	}
}

func (handle *Handle) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	handle.mu.RLock()
	defer handle.mu.RUnlock()

	if handle.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	response, err := handle.client.ReadDirAll(clientContext, &vfs_api.ReadDirAllRequest{
		Identifier: handle.directory.GetIdentifier(),
	})

	if err != nil {
		message := fmt.Sprintf("Failed to read directory %d", handle.directory.GetIdentifier())
		handle.logger.Error(message, err)
		return nil, err
	}

	var entries []fuse.Dirent

	for _, entry := range response.Nodes {
		switch entry.Type {
		case vfs_api.NodeType_FILE:
			entries = append(entries, fuse.Dirent{
				Name: entry.Name,
				Type: fuse.DT_File,
			})
		case vfs_api.NodeType_DIRECTORY:
			entries = append(entries, fuse.Dirent{
				Name: entry.Name,
				Type: fuse.DT_Dir,
			})
		}
	}

	// for _, tempFile := range handle.directory.tempFiles {
	// 	entries = append(entries, fuse.Dirent{
	// 		Name: tempFile.name,
	// 		Type: fuse.DT_File,
	// 	})
	// }

	return entries, nil
}

func (handle *Handle) Close() error {
	handle.mu.Lock()
	defer handle.mu.Unlock()

	if handle.IsClosed() {
		return nil
	}

	handle.cancel()

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

