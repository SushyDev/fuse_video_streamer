package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	directory_handle_service_factory "fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/handle/service/factory"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	directoryNodeService interfaces.DirectoryNodeService
	fileNodeService      interfaces.FileNodeService

	directoryHandleService interfaces.DirectoryHandleService
	client                 vfs_api.FileSystemServiceClient
	identifier             uint64

	// tempFiles []*TempFile

	logger *logger.Logger

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryNode = &Node{}

func New(
	directoryNodeService interfaces.DirectoryNodeService,
	fileNodeService interfaces.FileNodeService,
	client vfs_api.FileSystemServiceClient,
	logger *logger.Logger,
	identifier uint64,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		directoryNodeService:   directoryNodeService,

		fileNodeService: fileNodeService,

		client:     client,
		identifier: identifier,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}

	directoryHandleServiceFactory := directory_handle_service_factory.New()

	directoryHandleService, err := directoryHandleServiceFactory.New(node, client)
	if err != nil {
		panic(err)
	}

	node.directoryHandleService = directoryHandleService

	return node

}

func (fuseDirectory *Node) GetIdentifier() uint64 {
	return fuseDirectory.identifier
}

func (fuseDirectory *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
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

func (fuseDirectory *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	return fuseDirectory.directoryHandleService.New()
}

func (fuseDirectory *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30*time.Second)
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

		return fuseDirectory.fileNodeService.New(response.Node.Identifier, sizeResponse.Size)
	case vfs_api.NodeType_DIRECTORY:
		return fuseDirectory.directoryNodeService.New(response.Node.Identifier)
	}

	// for _, tempFile := range fuseDirectory.tempFiles {
	// 	if tempFile.name == lookupRequest.Name {
	// 		return tempFile, nil
	// 	}
	// }

	return nil, syscall.ENOENT
}

func (fuseDirectory *Node) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30*time.Second)
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

func (fuseDirectory *Node) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fs.Node) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	newDirectory, ok := newDir.(*Node)
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

func (fuseDirectory *Node) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, nil, syscall.ENOENT
	}

	// Disabled for now
	return nil, nil, syscall.ENOSYS

	// fuseDirectory.logger.Info(fmt.Sprintf("Create: %s", request.Name))
	//
	// tempFile := NewTempFile(request.Name, 0)
	//
	// fuseDirectory.tempFiles = append(fuseDirectory.tempFiles, tempFile)
	//
	// return tempFile, nil, nil
}

func (fuseDirectory *Node) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30*time.Second)
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

	return fuseDirectory.directoryNodeService.New(response.Node.Identifier)
}

func (fuseDirectory *Node) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	oldFile := oldNode.(interfaces.FileNode)

	_, err := fuseDirectory.client.Link(clientContext, &vfs_api.LinkRequest{
		Identifier:       oldFile.GetIdentifier(),
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

func (fuseDirectory *Node) Close() error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	if fuseDirectory.IsClosed() {
		return nil
	}

	fuseDirectory.cancel()

	fmt.Println("Closing directory", fuseDirectory.identifier)

	// for _, tempFile := range fuseDirectory.tempFiles {
	// 	err := tempFile.Close()
	// 	if err != nil {
	// 		fuseDirectory.logger.Error("Failed to close temp file", err)
	// 	}
	// }

	return nil
}

func (fuseDirectory *Node) IsClosed() bool {
	select {
	case <-fuseDirectory.ctx.Done():
		return true
	default:
		return false
	}
}
