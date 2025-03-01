package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/fuse/interfaces"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Directory struct {
	nodeService interfaces.NodeService
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	tempFiles []*TempFile

	logger *logger.Logger

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.Directory = &Directory{}

func NewDirectory(nodeService interfaces.NodeService, client vfs_api.FileSystemServiceClient, logger *logger.Logger, identifier uint64) *Directory {
	ctx, cancel := context.WithCancel(context.Background())

	return &Directory{
		nodeService: nodeService,
		client:     client,
		identifier: identifier,

		logger:     logger,

		mu:        sync.RWMutex{},

		ctx: ctx,
		cancel: cancel,
	}
}

func (fuseDirectory *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
	if fuseDirectory.IsClosed() {
		return syscall.ENOENT
	}

	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	attr.Mode = os.ModeDir | 0o777

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (fuseDirectory *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	return fuseDirectory, nil
}

func (fuseDirectory *Directory) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	response, err := fuseDirectory.client.Lookup(clientContext, &vfs_api.LookupRequest{
		Identifier: fuseDirectory.identifier,
		Name:       lookupRequest.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", lookupRequest.Name)
		fuseDirectory.logger.Error(message, err)
		return nil, syscall.ENOENT
	}

	if response.Node == nil {
		return nil, syscall.ENOENT
	}

	switch response.Node.Type {
	case vfs_api.NodeType_FILE:
		sizeResponse, err := fuseDirectory.client.GetVideoSize(ctx, &vfs_api.GetVideoSizeRequest{
			Identifier: response.Node.Identifier,
		})

		if err != nil {
			message := fmt.Sprintf("Failed to get video size for %s", lookupRequest.Name)
			fuseDirectory.logger.Error(message, err)
			return nil, syscall.ENOENT
		}

		return fuseDirectory.nodeService.NewFile(fuseDirectory.client, response.Node.Identifier, sizeResponse.Size)
	case vfs_api.NodeType_DIRECTORY:
		return fuseDirectory.nodeService.NewDirectory(fuseDirectory.client, response.Node.Identifier)
	}

	for _, tempFile := range fuseDirectory.tempFiles {
		if tempFile.name == lookupRequest.Name {
			return tempFile, nil
		}
	}

	return nil, syscall.ENOENT
}

func (fuseDirectory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	response, err := fuseDirectory.client.ReadDirAll(clientContext, &vfs_api.ReadDirAllRequest{
		Identifier: fuseDirectory.identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to read directory %d", fuseDirectory.identifier)
		fuseDirectory.logger.Error(message, err)
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

	for _, tempFile := range fuseDirectory.tempFiles {
		entries = append(entries, fuse.Dirent{
			Name: tempFile.name,
			Type: fuse.DT_File,
		})
	}

	return entries, nil
}

func (fuseDirectory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	_, err := fuseDirectory.client.Remove(clientContext, &vfs_api.RemoveRequest{
		Identifier: fuseDirectory.identifier,
		Name:       removeRequest.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to remove %s", removeRequest.Name)
		fuseDirectory.logger.Error(message, err)
		return err
	}

	return nil
}

func (fuseDirectory *Directory) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fs.Node) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	newDirectory, ok := newDir.(*Directory)
	if !ok {
		return syscall.ENOSYS
	}


	_, err := fuseDirectory.client.Rename(clientContext, &vfs_api.RenameRequest{
		ParentIdentifier:    fuseDirectory.identifier,
		Name:                request.OldName,
		NewName:             request.NewName,
		NewParentIdentifier: newDirectory.identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to rename %s to %s", request.OldName, request.NewName)
		fuseDirectory.logger.Error(message, err)
		return err
	}

	return nil
}

func (fuseDirectory *Directory) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, nil, syscall.ENOENT
	}

	// Disabled for now
	return nil, nil, syscall.ENOSYS

	fuseDirectory.logger.Info(fmt.Sprintf("Create: %s", request.Name))

	tempFile := NewTempFile(request.Name, 0)

	fuseDirectory.tempFiles = append(fuseDirectory.tempFiles, tempFile)

	return tempFile, tempFile, nil
}

func (fuseDirectory *Directory) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	response, err := fuseDirectory.client.Mkdir(clientContext, &vfs_api.MkdirRequest{
		ParentIdentifier: fuseDirectory.identifier,
		Name:             request.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to mkdir %s", request.Name)
		fuseDirectory.logger.Error(message, err)
		return nil, err
	}

	return fuseDirectory.nodeService.NewDirectory(fuseDirectory.client, response.Node.Identifier)
}

func (fuseDirectory *Directory) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()

	oldFile := oldNode.(*File)

	_, err := fuseDirectory.client.Link(clientContext, &vfs_api.LinkRequest{
		Identifier:       oldFile.identifier,
		ParentIdentifier: fuseDirectory.identifier,
		Name:             request.NewName,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to link %s", request.NewName)
		fuseDirectory.logger.Error(message, err)
		return nil, err
	}

	return oldFile, nil
}

func (fuseDirectory *Directory) Close() error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil
	}

	fuseDirectory.cancel()

	// for _, tempFile := range fuseDirectory.tempFiles {
	// 	err := tempFile.Close()
	// 	if err != nil {
	// 		fuseDirectory.logger.Error("Failed to close temp file", err)
	// 	}
	// }

	return nil
}

func (fuseDirectory *Directory) IsClosed() bool {
	select {
	case <-fuseDirectory.ctx.Done():
		return true
	default:
		return false
	}
}
