package node

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	client        interfaces_filesystem_client.Client
	loggerFactory interfaces_logger.LoggerFactory

	handleService interfaces_fuse.FileHandleService

	metrics *metrics.FileNodeMetrics
	logger  interfaces_logger.Logger

	identifier uint64
	size       uint64

	handles []interfaces_fuse.FileHandle

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.FileNode = &Node{}

func New(
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
	fileHandleService interfaces_fuse.FileHandleService,
	metric *metrics.FileNodeMetrics,
	logger interfaces_logger.Logger,
	identifier uint64,
	size uint64,
) *Node {
	node := &Node{
		client:        client,
		loggerFactory: loggerFactory,

		handleService: fileHandleService,

		metrics: metric,
		logger:  logger,

		identifier: identifier,
		size:       size,
	}

	return node
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
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
		node.logger.Warn("Node is closed, cannot open file handle")
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	handle, err := node.handleService.New(node)
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
	node.handleService = nil

	var wg sync.WaitGroup

	for _, handle := range node.handles {
		wg.Add(1)

		go func() {
			defer wg.Done()
			handle.Close()
			handle = nil
		}()
	}

	wg.Wait()

	node.handles = nil

	return nil
}

func (node *Node) IsClosed() bool {
	return node.closed.Load()
}
