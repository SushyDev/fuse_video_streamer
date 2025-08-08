package directory

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	handle_directory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/directory"
	node_symlink "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/symlink"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	client           interfaces_filesystem_client.Client
	identifier       uint64
	remoteIdentifier uint64

	logger interfaces_logger.Logger

	directoryHandleService interfaces_handle.DirectoryHandleService
	directoryNodeService   interfaces_node.DirectoryNodeService
	streamableNodeService  interfaces_node.StreamableNodeService
	fileNodeService        interfaces_node.FileNodeService
	loggerFactory          interfaces_logger.LoggerFactory

	handles []interfaces_handle.DirectoryHandle

	mu     sync.RWMutex
	closed atomic.Bool
}

var _ interfaces_node.DirectoryNode = &Node{}

func NewNode(
	abstractNode interfaces_node.AbstractNode,
	logger interfaces_logger.Logger,

	loggerFactory interfaces_logger.LoggerFactory,
	directoryNodeService interfaces_node.DirectoryNodeService,
	streamableNodeService interfaces_node.StreamableNodeService,
	fileNodeService interfaces_node.FileNodeService,
) (*Node, error) {
	directoryHandleServiceFactory := handle_directory.NewFactory(loggerFactory)

	node := &Node{
		client:           abstractNode.GetClient(),
		identifier:       abstractNode.GetIdentifier(),
		remoteIdentifier: abstractNode.GetRemoteIdentifier(),

		directoryHandleService: directoryHandleServiceFactory.NewService(),
		directoryNodeService:   directoryNodeService,
		streamableNodeService:  streamableNodeService,
		fileNodeService:        fileNodeService,
		loggerFactory:          loggerFactory,

		logger: logger,
	}

	return node, nil
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetRemoteIdentifier() uint64 {
	return node.remoteIdentifier
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

	handle, err := node.directoryHandleService.NewHandle(node)
	if err != nil {
		message := "failed to open directory"
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

	switch lookupRequest.Name {
	case "", ".":
		node.logger.Debug("lookup request for current directory or empty name, returning self")
		return node, nil
	case "..":
		node.logger.Debug("lookup request for parent directory, returning parent node")
		parentNode, err := node.directoryNodeService.NewNode(node, node.GetRemoteIdentifier())
		if err != nil {
			node.logger.Error("failed to create parent node", err)
			return nil, err
		}
		return parentNode, nil
	}

	client_filesystem := node.client.GetFileSystem()
	foundNode, err := client_filesystem.Lookup(node.GetRemoteIdentifier(), lookupRequest.Name)

	if err == syscall.ENOENT {
		// node.logger.Error(fmt.Sprintf("node: %s not found in directory with ID: %d", lookupRequest.Name, node.GetRemoteIdentifier()), nil)

		return nil, syscall.ENOENT
	} else if err != nil {
		node.logger.Error(fmt.Sprintf("failed to lookup node: %s in directory with ID: %d", lookupRequest.Name, node.GetRemoteIdentifier()), err)

		return nil, syscall.EAGAIN
	}

	if foundNode == nil {
		// node.logger.Error(fmt.Sprintf("node: %s not found in directory with ID: %d", lookupRequest.Name, node.GetRemoteIdentifier()), nil)
		return nil, syscall.ENOENT
	}

	// node.logger.Debug(fmt.Sprintf("found node: %s with ID: %d in directory with ID: %d", foundNode.GetName(), foundNode.GetId(), node.GetRemoteIdentifier()))

	switch foundNode.GetMode() {
	case io_fs.ModeDir:
		return node.directoryNodeService.NewNode(node, foundNode.GetId())
	case io_fs.FileMode(0):
		if foundNode.GetStreamable() {
			return node.streamableNodeService.NewNode(node, foundNode.GetId())
		} else {
			return node.fileNodeService.NewNode(node, foundNode.GetId())
		}
	case io_fs.ModeSymlink:
		symlinkLogger, err := node.loggerFactory.NewLogger("Symlink node")
		if err != nil {
			node.logger.Error("failed to create logger for symlink node", err)
			return nil, err
		}

		return node_symlink.NewNode(node.client, symlinkLogger, foundNode.GetId()), nil
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

	err := fileSystem.Remove(node.GetRemoteIdentifier(), removeRequest.Name)
	if err != nil {
		message := fmt.Sprintf("failed to remove %s", removeRequest.Name)
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

	err := fileSystem.Rename(node.GetRemoteIdentifier(), request.OldName, newDirectory.GetRemoteIdentifier(), request.NewName)
	if err != nil {
		message := fmt.Sprintf("failed to rename %s to %s", request.OldName, request.NewName)
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

	err := fileSystem.Create(node.GetRemoteIdentifier(), request.Name, io_fs.FileMode(request.Mode))
	if err != nil {
		message := fmt.Sprintf("failed to create %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	foundNode, err := fileSystem.Lookup(node.GetRemoteIdentifier(), request.Name)
	if err != nil {
		message := fmt.Sprintf("failed to lookup %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	fileNode, err := node.fileNodeService.NewNode(node, foundNode.GetId())
	if err != nil {
		message := fmt.Sprintf("failed to create file node %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	handle, err := fileNode.Open(ctx, &fuse.OpenRequest{}, &fuse.OpenResponse{})
	if err != nil {
		message := fmt.Sprintf("failed to open file node %s", request.Name)
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

	remoteDirectoryNode, err := fileSystem.MkDir(node.GetRemoteIdentifier(), request.Name)
	if err != nil {
		message := fmt.Sprintf("failed to mkdir %s", request.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	return node.directoryNodeService.NewNode(node, remoteDirectoryNode.GetId())
}

func (node *Node) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	oldFile, ok := oldNode.(interfaces_node.StreamableNode)
	if !ok {
		message := fmt.Sprintf("not a streamable node: %s", oldNode)
		node.logger.Error(message, nil)
		return nil, syscall.ENOSYS
	}

	fileSystem := node.client.GetFileSystem()

	err := fileSystem.Link(node.GetRemoteIdentifier(), request.NewName, oldFile.GetRemoteIdentifier())
	if err != nil {
		message := fmt.Sprintf("failed to link %s", request.NewName)
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
