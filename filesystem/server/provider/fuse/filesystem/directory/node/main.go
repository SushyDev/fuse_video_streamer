package node

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"os"
	"sync"
	"syscall"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	directory_handle_service_factory "fuse_video_streamer/filesystem/server/provider/fuse/filesystem/directory/handle/service/factory"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/symlink"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	directoryHandleService interfaces.DirectoryHandleService

	directoryNodeService  interfaces.DirectoryNodeService
	streamableNodeService interfaces.StreamableNodeService
	fileNodeService       interfaces.FileNodeService

	client     filesystem_client_interfaces.Client
	identifier uint64

	handles []interfaces.DirectoryHandle

	logger *logger.Logger

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryNode = &Node{}

func New(
	directoryNodeService interfaces.DirectoryNodeService,
	streamableNodeService interfaces.StreamableNodeService,
	fileNodeService interfaces.FileNodeService,
	client filesystem_client_interfaces.Client,
	logger *logger.Logger,
	identifier uint64,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		directoryNodeService:  directoryNodeService,
		streamableNodeService: streamableNodeService,
		fileNodeService:       fileNodeService,

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

	attr.Mode = os.ModeDir

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

	fileSystem := node.client.GetFileSystem()

	fmt.Printf("Looking up node: %s in directory with ID: %d\n", lookupRequest.Name, node.GetIdentifier())

	foundNode, err := fileSystem.Lookup(node.GetIdentifier(), lookupRequest.Name)
	if err != nil {
		node.logger.Error(fmt.Sprintf("Failed to lookup node: %s in directory with ID: %d, error: %v", lookupRequest.Name, node.GetIdentifier(), err))

		return nil, syscall.EAGAIN
	}

	if foundNode == nil {
		return nil, syscall.ENOENT
	}

	fmt.Printf("Found node: %s with ID: %d\n", foundNode.GetName(), foundNode.GetId())

	switch foundNode.GetMode() {
	case io_fs.ModeDir:
		return node.directoryNodeService.New(foundNode.GetId())
	case io_fs.FileMode(0):
		if foundNode.GetStreamable() {
			return node.streamableNodeService.New(foundNode.GetId())
		} else {
			return node.fileNodeService.New(foundNode.GetId())
		}
	case io_fs.ModeSymlink:
		return symlink.New(node.client, foundNode.GetId()), nil
	default:
		message := fmt.Sprintf("Unknown file mode: %s", foundNode.GetName())
		node.logger.Error(message, nil)
		return nil, syscall.ENOENT
	}
}

func (node *Node) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return syscall.ENOENT
	}

	fileSystem := node.client.GetFileSystem()

	err := fileSystem.Remove(node.identifier, removeRequest.Name)
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

	newDirectory, ok := newDir.(*Node)
	if !ok {
		return syscall.ENOSYS
	}

	fileSystem := node.client.GetFileSystem()

	err := fileSystem.Rename(node.GetIdentifier(), request.OldName, newDirectory.GetIdentifier(), request.NewName)
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

	fileSystem := node.client.GetFileSystem()

	err := fileSystem.Create(node.GetIdentifier(), request.Name, io_fs.FileMode(request.Mode))
	if err != nil {
		message := fmt.Sprintf("Failed to create %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	foundNode, err := fileSystem.Lookup(node.GetIdentifier(), request.Name)
	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	fileNode, err := node.fileNodeService.New(foundNode.GetId())
	if err != nil {
		message := fmt.Sprintf("Failed to create file node %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	handle, err := fileNode.Open(ctx, &fuse.OpenRequest{}, &fuse.OpenResponse{})
	if err != nil {
		message := fmt.Sprintf("Failed to open file node %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	return fileNode, handle, nil
}

func (node *Node) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	fileSystem := node.client.GetFileSystem()

	newDir, err := fileSystem.MkDir(node.GetIdentifier(), request.Name)
	if err != nil {
		message := fmt.Sprintf("Failed to mkdir %s", request.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	return node.directoryNodeService.New(newDir.GetId())
}

func (node *Node) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	oldFile, ok := oldNode.(interfaces.StreamableNode)
	if !ok {
		message := fmt.Sprintf("Not a streamable node: %s", oldNode)
		node.logger.Error(message, nil)
		return nil, syscall.ENOSYS
	}

	fileSystem := node.client.GetFileSystem()

	err := fileSystem.Link(node.GetIdentifier(), request.NewName, oldFile.GetIdentifier())
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
