package node

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_directory_handle_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle/service/factory"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/symlink"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	directoryHandleService interfaces_fuse.DirectoryHandleService

	directoryNodeService  interfaces_fuse.DirectoryNodeService
	streamableNodeService interfaces_fuse.StreamableNodeService
	fileNodeService       interfaces_fuse.FileNodeService

	loggerFactory interfaces_logger.LoggerFactory

	client     interfaces_filesystem_client.Client
	identifier uint64

	handles []interfaces_fuse.DirectoryHandle

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.DirectoryNode = &Node{}

func New(
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
	directoryNodeService interfaces_fuse.DirectoryNodeService,
	streamableNodeService interfaces_fuse.StreamableNodeService,
	fileNodeService interfaces_fuse.FileNodeService,
	logger interfaces_logger.Logger,
	identifier uint64,
) (*Node, error) {
	directoryHandleServiceFactory := factory_directory_handle_service.New(loggerFactory)

	node := &Node{
		directoryHandleService: directoryHandleServiceFactory.New(),

		directoryNodeService:  directoryNodeService,
		streamableNodeService: streamableNodeService,
		fileNodeService:       fileNodeService,

		loggerFactory: loggerFactory,

		client:     client,
		identifier: identifier,

		logger: logger,
	}

	return node, nil

}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetClient() interfaces_filesystem_client.Client {
	return node.client
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	attr.Mode = os.ModeDir

	return nil
}

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	handle, err := node.directoryHandleService.New(node)
	if err != nil {
		message := "Failed to open directory"
		node.logger.Error(message, err)
		return nil, err
	}

	node.handles = append(node.handles, handle)

	return handle, nil
}

func (node *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	client_filesystem := node.client.GetFileSystem()
	foundNode, err := client_filesystem.Lookup(node.GetIdentifier(), lookupRequest.Name)

	if err == syscall.ENOENT {
		return nil, syscall.ENOENT
	} else if err != nil {
		node.logger.Error(fmt.Sprintf("Failed to lookup node: %s in directory with ID: %d", lookupRequest.Name, node.GetIdentifier()), err)

		return nil, syscall.EAGAIN
	}

	if foundNode == nil {
		return nil, syscall.ENOENT
	}

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
		symlinkLogger, err := node.loggerFactory.NewLogger("Symlink node")
		if err != nil {
			node.logger.Error("Failed to create logger for symlink node", err)
			return nil, err
		}

		return symlink.New(node.client, symlinkLogger, foundNode.GetId()), nil
	default:
		message := fmt.Sprintf("Unknown file mode: %s", foundNode.GetName())
		node.logger.Error(message, nil)
		return nil, syscall.ENOENT
	}
}

func (node *Node) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

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
	if node.IsClosed() {
		return syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

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
	if node.IsClosed() {
		return nil, nil, syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

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
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

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
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	oldFile, ok := oldNode.(interfaces_fuse.StreamableNode)
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
	if !node.closed.CompareAndSwap(false, true) {
		return nil
	}

	for _, handle := range node.handles {
		handle.Close()
		handle = nil
	}

	return nil
}

func (node *Node) IsClosed() bool {
	return node.closed.Load()
}
