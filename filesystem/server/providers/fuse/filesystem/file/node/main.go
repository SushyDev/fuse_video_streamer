package node

import (
	"context"
	"os"
	"sync"
	"syscall"

	file_handle_service_factory "fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/handle/service/factory"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream/factory"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	streamFactory *factory.Factory
	fileHandleService interfaces.FileHandleService
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	size    uint64

	id string

	logger *logger.Logger

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc

	handles map[uint64]interfaces.FileHandle
}

func New(client vfs_api.FileSystemServiceClient, logger *logger.Logger, identifier uint64, size uint64) *Node {
	context, cancel := context.WithCancel(context.Background())

	stream_factory := factory.NewFactory(client, identifier, size)

	fileHandleServiceFactory := file_handle_service_factory.New()

	node := &Node{
		streamFactory: stream_factory,
		client:     client,
		identifier: identifier,
	
		size: size,

		logger: logger,

		mu: sync.RWMutex{},

		ctx: context,
		cancel: cancel,

		handles: make(map[uint64]interfaces.FileHandle),
	}

	fileHandleService, err := fileHandleServiceFactory.New(node, client)
	if err != nil {
		panic(err)
	}

	node.fileHandleService = fileHandleService

	return node
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetSize() uint64 {
	return node.size
}

func (node *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.isClosed() {
		return syscall.ENOENT
	}

	attr.Inode = node.identifier
	attr.Mode = os.ModePerm | 0o777
	attr.Size = node.size

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

	openResponse.Flags |= fuse.OpenDirectIO

	handle, err := node.fileHandleService.New()
	if err != nil {
		message := "Failed to create file handle"
		node.logger.Error(message, err)
		return nil, err
	}

	node.handles[handle.GetIdentifier()] = handle

	return handle, nil
}

func (node *Node) Close() error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.isClosed() {
		return nil
	}

	node.cancel()

	for identifier, handle := range node.handles {
		handle.Close()

		delete(node.handles, identifier)
	}

	node.handles = nil

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
