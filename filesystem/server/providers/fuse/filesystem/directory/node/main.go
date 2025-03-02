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
	directoryHandleService interfaces.DirectoryHandleService
	directoryNodeService interfaces.DirectoryNodeService
	fileNodeService      interfaces.FileNodeService

	client                 vfs_api.FileSystemServiceClient
	identifier             uint64

	// tempFiles []*TempFile
	handles []interfaces.DirectoryHandle

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

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return syscall.ENOENT
	}

	attr.Mode = os.ModeDir | 0o777

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	handle, err := node.directoryHandleService.New()
	if err != nil {
		message := "Failed to open directory"
		node.logger.Error(message, err)
		return nil, err
	}

	node.handles = append(node.handles, handle)

	return handle, nil
}

func (node *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	response, err := node.client.Lookup(clientContext, &vfs_api.LookupRequest{
		Identifier: node.identifier,
		Name:       lookupRequest.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", lookupRequest.Name)
		node.logger.Error(message, err)
		return nil, syscall.ENOENT
	}

	if response.Node == nil {
		return nil, syscall.ENOENT
	}

	switch response.Node.Type {
	case vfs_api.NodeType_FILE:
		clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
		defer cancel()

		sizeResponse, err := node.client.GetVideoSize(clientContext, &vfs_api.GetVideoSizeRequest{
			Identifier: response.Node.Identifier,
		})

		if err != nil {
			message := fmt.Sprintf("Failed to get video size for %s", lookupRequest.Name)
			node.logger.Error(message, err)
			return nil, syscall.ENOENT
		}

		return node.fileNodeService.New(response.Node.Identifier, sizeResponse.Size)
	case vfs_api.NodeType_DIRECTORY:
		return node.directoryNodeService.New(response.Node.Identifier)
	}

	return nil, syscall.ENOENT
}

func (node *Node) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	_, err := node.client.Remove(clientContext, &vfs_api.RemoveRequest{
		Identifier: node.identifier,
		Name:       removeRequest.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to remove %s", removeRequest.Name)
		node.logger.Error(message, err)
		return err
	}

	return nil
}

func (node *Node) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fs.Node) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	newDirectory, ok := newDir.(*Node)
	if !ok {
		return syscall.ENOSYS
	}

	_, err := node.client.Rename(clientContext, &vfs_api.RenameRequest{
		ParentIdentifier:    node.identifier,
		Name:                request.OldName,
		NewName:             request.NewName,
		NewParentIdentifier: newDirectory.identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to rename %s to %s", request.OldName, request.NewName)
		node.logger.Error(message, err)
		return err
	}

	return nil
}

func (node *Node) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
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

func (node *Node) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	response, err := node.client.Mkdir(clientContext, &vfs_api.MkdirRequest{
		ParentIdentifier: node.identifier,
		Name:             request.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to mkdir %s", request.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	return node.directoryNodeService.New(response.Node.Identifier)
}

func (node *Node) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	oldFile := oldNode.(interfaces.FileNode)

	_, err := node.client.Link(clientContext, &vfs_api.LinkRequest{
		Identifier:       oldFile.GetIdentifier(),
		ParentIdentifier: node.identifier,
		Name:             request.NewName,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to link %s", request.NewName)
		node.logger.Error(message, err)
		return nil, err
	}

	return oldFile, nil
}

func (node *Node) Close() error {
	// node.mu.Lock()
	// defer node.mu.Unlock()

	if node.isClosed() {
		return nil
	}

	node.cancel()

	for _, handle := range node.handles {
		handle.Close()
		handle = nil
	}

	return nil
}

func (node *Node) isClosed() bool {
	select {
	case <-node.ctx.Done():
		return true
	default:
		return false
	}
}
