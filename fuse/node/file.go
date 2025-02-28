package node

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream/manager"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.Handle = &File{}

type File struct {
	stream_manager *stream_manager.Manager
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	size    uint64

	logger *logger.Logger

	mu sync.RWMutex
}

func NewFile(client vfs_api.FileSystemServiceClient, identifier uint64, size uint64) *File {
	logger, err := logger.NewLogger("File Node")
	if err != nil {
		panic(err)
	}

	stream_manager := stream_manager.NewManager(client, identifier, size)

	return &File{
		stream_manager: stream_manager,
		client:     client,
		identifier: identifier,

		size: size,

		logger: logger,
	}
}

var _ fs.Node = &File{}

func (fuseFile *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = fuseFile.identifier
	attr.Mode = os.ModePerm | 0o777
	attr.Size = fuseFile.size

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

// var _ fs.NodeOpener = &File{}
//
// func (fuseFile *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
// 	fuseFile.mu.RLock()
// 	defer fuseFile.mu.RUnlock()
//
// 	// openResponse.Flags |= fuse.OpenKeepCache
//
// 	return fuseFile, nil
// }

var _ fs.HandleReader = &File{}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	videoStream, err := fuseFile.stream_manager.GetOrCreateStream(readRequest.Pid)
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

		fuseFile.stream_manager.DeleteStream(readRequest.Pid)

		return err
	}
}

var _ fs.HandleFlusher = &File{}

func (fuseFile *File) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	fuseFile.stream_manager.DeleteStream(flushRequest.Pid)

	return nil
}
