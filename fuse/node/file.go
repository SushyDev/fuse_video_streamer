package node

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream/factory"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type File struct {
	streamFactory *factory.Factory
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	size    uint64

	id string

	logger *logger.Logger

	mu sync.RWMutex

	context context.Context
	cancel context.CancelFunc
}

func NewFile(client vfs_api.FileSystemServiceClient, logger *logger.Logger, stream_factory *factory.Factory, identifier uint64, size uint64) *File {
	context, cancel := context.WithCancel(context.Background())

	return &File{
		streamFactory: stream_factory,
		client:     client,
		identifier: identifier,
	
		size: size,

		logger: logger,

		mu: sync.RWMutex{},

		context: context,
		cancel: cancel,
	}
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

func (fuseFile *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	if fuseFile.IsClosed() {
		return nil, syscall.ENOENT
	}

	// openResponse.Flags |= fuse.OpenKeepCache

	uniq := time.Now().UnixNano()

	fuseFile.id = fmt.Sprintf("%d-%d", fuseFile.identifier, uniq)

	// todo implement fuse in a shallow file and return a file handle with the stream, this way we can avoid using PID per stream and use a generated uniq id instead
	// this will be better also for docker containers so we dont need pid namespace sharing

	return fuseFile, nil
}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	if fuseFile.IsClosed() {
		return syscall.ENOENT
	}

	videoStream, err := fuseFile.streamFactory.GetOrCreateStream(readRequest.Pid)
	if err != nil {
		message := "Failed to get or create video stream"
		fuseFile.logger.Error(message, err)
		return fmt.Errorf(message)
	}

	// TODO buffer pool
	buffer := make([]byte, readRequest.Size)

	bytesRead, err := videoStream.ReadAt(buffer, readRequest.Offset)
	switch err {
	case nil:
		readResponse.Data = buffer[:bytesRead]
		return nil

	case io.EOF:
		readResponse.Data = buffer[:bytesRead]
		return nil

	default:
		message := fmt.Sprintf("Failed to read video stream for pid %d, closing video stream", readRequest.Pid)
		fuseFile.logger.Error(message, err)

		fuseFile.streamFactory.DeleteStream(readRequest.Pid)

		return err
	}
}

func (fuseFile *File) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	fuseFile.streamFactory.DeleteStream(flushRequest.Pid)

	return nil
}

func (fuseFile *File) Close() error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	if fuseFile.IsClosed() {
		return nil
	}

	fuseFile.streamFactory.Close()

	fuseFile.cancel()

	return nil
}

func (fuseFile *File) IsClosed() bool {
	select {
	case <-fuseFile.context.Done():
		return true
	default:
		return false
	}
}

