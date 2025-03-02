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

type File struct {
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

func New(client vfs_api.FileSystemServiceClient, logger *logger.Logger, identifier uint64, size uint64) *File {
	context, cancel := context.WithCancel(context.Background())

	stream_factory := factory.NewFactory(client, identifier, size)

	fileHandleServiceFactory := file_handle_service_factory.New()

	node := &File{
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

func (fuseFile *File) GetIdentifier() uint64 {
	return fuseFile.identifier
}

func (fuseFile *File) GetSize() uint64 {
	return fuseFile.size
}

func (fuseFile *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	if fuseFile.IsClosed() {
		return syscall.ENOENT
	}

	attr.Inode = fuseFile.identifier
	attr.Mode = os.ModePerm | 0o777
	attr.Size = fuseFile.size

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

var _ fs.NodeOpener = &File{}

func (file *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	file.mu.RLock()
	defer file.mu.RUnlock()

	if file.IsClosed() {
		return nil, syscall.ENOENT
	}

	// todo since using direct io to prevent caching handles i should also implement some LRU cache in front of the stream
	openResponse.Flags |= fuse.OpenDirectIO

	handle, err := file.fileHandleService.New()
	if err != nil {
		return nil, err
	}

	file.handles[handle.GetIdentifier()] = handle

	return handle, nil
}

func (fuseFile *File) Close() error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	if fuseFile.IsClosed() {
		return nil
	}

	fuseFile.cancel()

	for _, handle := range fuseFile.handles {
		handle.Close()

		delete(fuseFile.handles, handle.GetIdentifier())
	}

	return nil
}

func (fuseFile *File) IsClosed() bool {
	select {
	case <-fuseFile.ctx.Done():
		return true
	default:
		return false
	}
}
