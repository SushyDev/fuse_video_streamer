package streamable

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	handleService handle.StreamableHandleService

	client           interfaces_filesystem_client.Client
	identifier       uint64
	remoteIdentifier uint64
	size             uint64

	handles []handle.StreamableHandle

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ node.StreamableNode = &Node{}

func NewNode(
	client interfaces_filesystem_client.Client,
	streamableHandleServiceFactory handle.StreamableHandleServiceFactory,
	logger interfaces_logger.Logger,
	identifier uint64,
	remoteIdentifier uint64,
	size uint64,
) (*Node, error) {
	node := &Node{
		client: client,
		logger: logger,

		identifier:       identifier,
		remoteIdentifier: remoteIdentifier,
		size:             size,
	}

	fileHandleService, err := streamableHandleServiceFactory.NewService(node, client)
	if err != nil {
		node.logger.Error("failed to create file handle service", err)
		return nil, err
	}

	node.handleService = fileHandleService

	return node, nil
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetRemoteIdentifier() uint64 {
	return node.remoteIdentifier
}

func (node *Node) GetSize() uint64 {
	return node.size
}

func (node *Node) GetClient() interfaces_filesystem_client.Client {
	return node.client
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	attr.Mode = os.FileMode(0)
	attr.Size = node.size

	return nil
}

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	handle, err := node.handleService.NewHandle()
	if err != nil {
		message := "failed to create file handle"
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

func (node *Node) IsClosed() bool {
	return node.closed.Load()
}
