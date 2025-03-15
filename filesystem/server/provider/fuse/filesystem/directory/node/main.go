package node

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"time"

	directory_handle_service_factory "fuse_video_steamer/filesystem/server/provider/fuse/filesystem/directory/handle/service/factory"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"

	"github.com/anacrolix/fuse"
	fuse_fs "github.com/anacrolix/fuse/fs"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Node struct {
	directoryHandleService interfaces.DirectoryHandleService

	directoryNodeService   interfaces.DirectoryNodeService
	streamableNodeService interfaces.StreamableNodeService
	fileNodeService        interfaces.FileNodeService

	client     api.FileSystemServiceClient
	identifier uint64

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
	streamableNodeService interfaces.StreamableNodeService,
	fileNodeService interfaces.FileNodeService,
	client api.FileSystemServiceClient,
	logger *logger.Logger,
	identifier uint64,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())

	node := &Node{
		directoryNodeService: directoryNodeService,
		streamableNodeService: streamableNodeService,
		fileNodeService:      fileNodeService,

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

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fuse_fs.Handle, error) {
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

func (node *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fuse_fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	response, err := node.client.Lookup(clientContext, &api.LookupRequest{
		NodeId: node.identifier,
		Name:   lookupRequest.Name,
	})

	if err != nil {
		status, ok := status.FromError(err)
		if ok && status.Code() == codes.NotFound {
			return nil, syscall.ENOENT
		}

		message := fmt.Sprintf("abc Failed to lookup %s", lookupRequest.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	if response.Node == nil {
		return nil, syscall.ENOENT
	}

	switch response.Node.GetMode() {
	case uint32(fs.ModeDir):
		return node.directoryNodeService.New(response.Node.GetId())
	case uint32(0):
		if response.Node.GetStreamable() {
			return node.streamableNodeService.New(response.Node.GetId())
		} else {
			return node.fileNodeService.New(response.Node.GetId())
		}
	default:
		return nil, syscall.ENOENT
	}
}

func (node *Node) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	_, err := node.client.Remove(clientContext, &api.RemoveRequest{
		ParentNodeId: node.identifier,
		Name:         removeRequest.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to remove %s", removeRequest.Name)
		node.logger.Error(message, err)
		return err
	}

	return nil
}

func (node *Node) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fuse_fs.Node) error {
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

	_, err := node.client.Rename(clientContext, &api.RenameRequest{
		NodeId:       node.identifier,
		ParentNodeId: newDirectory.identifier,
		Name:         request.OldName,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to rename %s to %s", request.OldName, request.NewName)
		node.logger.Error(message, err)
		return err
	}

	return nil
}

func (node *Node) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fuse_fs.Node, fuse_fs.Handle, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	_, err := node.client.Create(clientContext, &api.CreateRequest{
		ParentNodeId: node.identifier,
		Name:         request.Name,
		Mode:         uint32(0),
	})

	if err != nil {
		message := fmt.Sprintf("Failed to create %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	lookupResponse, err := node.client.Lookup(clientContext, &api.LookupRequest{
		NodeId: node.identifier,
		Name:   request.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", request.Name)
		node.logger.Error(message, err)
		return nil, nil, err
	}

	streamableNode, err := node.streamableNodeService.New(lookupResponse.Node.GetId())
	if err != nil {
		return nil, nil, err
	}

	var handle fuse_fs.Handle

	return streamableNode, handle, nil
}

func (node *Node) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fuse_fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	response, err := node.client.Mkdir(clientContext, &api.MkdirRequest{
		ParentNodeId: node.identifier,
		Name:         request.Name,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to mkdir %s", request.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	return node.directoryNodeService.New(response.Node.GetId())
}

func (node *Node) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fuse_fs.Node) (fuse_fs.Node, error) {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30*time.Second)
	defer cancel()

	oldFile := oldNode.(interfaces.StreamableNode)

	_, err := node.client.Link(clientContext, &api.LinkRequest{
		NodeId:       oldFile.GetIdentifier(),
		ParentNodeId: node.identifier,
		Name:         request.NewName,
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
