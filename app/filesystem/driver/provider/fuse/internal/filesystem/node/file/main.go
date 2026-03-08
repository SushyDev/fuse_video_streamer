package file

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	client        interfaces_filesystem_client.Client
	loggerFactory interfaces_logger.LoggerFactory

	handleService interfaces_handle.FileHandleService

	logger interfaces_logger.Logger

	identifier       uint64
	remoteIdentifier uint64
	size             uint64
	mode             os.FileMode

	handles []interfaces_handle.FileHandle

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_node.FileNode = &Node{}

func NewNode(
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
	fileHandleService interfaces_handle.FileHandleService,
	logger interfaces_logger.Logger,
	identifier uint64,
	remoteIdentifier uint64,
	size uint64,
	mode os.FileMode,
) *Node {
	node := &Node{
		client:        client,
		loggerFactory: loggerFactory,

		handleService: fileHandleService,

		logger: logger,

		identifier:       identifier,
		remoteIdentifier: remoteIdentifier,
		size:             size,
		mode:             mode,
	}

	return node
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

func (node *Node) GetMode() os.FileMode {
	return node.mode
}

func (node *Node) UpdateSize(newSize uint64) {
	node.mu.Lock()
	defer node.mu.Unlock()
	node.size = newSize
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	attr.Mode = node.mode
	attr.Size = node.size
	attr.Inode = node.remoteIdentifier
	attr.Valid = 30 * time.Second

	return nil
}

func (node *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	if node.IsClosed() {
		node.logger.Warn("Node is closed, cannot open file handle")
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	handle, err := node.handleService.NewHandle(node)
	if err != nil {
		message := "failed to create file handle"
		node.logger.Error(message, err)
		return nil, err
	}

	openResponse.Flags |= fuse.OpenDirectIO

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
