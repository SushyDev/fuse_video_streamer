package node

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	streamable_handle_service_factory "fuse_video_streamer/filesystem/server/provider/fuse/filesystem/streamable/handle/service/factory"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	handleService interfaces.StreamableHandleService

	client     filesystem_client_interfaces.Client
	identifier uint64
	size       uint64

	handles []interfaces.StreamableHandle

	logger *logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces.StreamableNode = &Node{}

func New(client filesystem_client_interfaces.Client, logger *logger.Logger, identifier uint64, size uint64) *Node {
	node := &Node{
		client:        client,
		identifier:    identifier,

		size: size,

		logger: logger,

		mu: sync.RWMutex{},
	}

	fileHandleServiceFactory := streamable_handle_service_factory.New()
	fileHandleService, err := fileHandleServiceFactory.New(node, client)
	if err != nil {
		panic(err)
	}

	node.handleService = fileHandleService

	return node
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetSize() uint64 {
	return node.size
}

func (node *Node) GetClient() filesystem_client_interfaces.Client {
	return node.client
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.isClosed() {
		return syscall.ENOENT
	}

	attr.Mode = os.FileMode(0)
	attr.Size = node.size

	return nil
}

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return nil, syscall.ENOENT
	}

	handle, err := node.handleService.New()
	if err != nil {
		message := "Failed to create file handle"
		node.logger.Error(message, err)
		return nil, err
	}

	node.handles = append(node.handles, handle)

	return handle, nil
}

func (node *Node) Close() error {
	if !node.closed.CompareAndSwap(false, true) {
		return nil
	}

	node.handleService.Close()

	for _, handle := range node.handles {
		handle.Close()
	}

	return nil
}

func (node *Node) isClosed() bool {
	return node.closed.Load()
}
