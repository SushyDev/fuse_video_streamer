package node

import (
	"context"
	"fmt"
	"os"
	"sync"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.Handle = &File{}

type File struct {
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	streams stream.Map
	size    uint64

	logger *logger.Logger

	mu sync.RWMutex
}

func NewFile(client vfs_api.FileSystemServiceClient, identifier uint64, size uint64) *File {
	logger, err := logger.NewLogger("File Node")
	if err != nil {
		panic(err)
	}

	return &File{
		client:     client,
		identifier: identifier,

		size: size,

		logger: logger,
	}
}

var _ fs.Node = &File{}

func (fuseFile *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

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

	openResponse.Flags |= fuse.OpenKeepCache

	return fuseFile, nil
}

var _ fs.HandleReader = &File{}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	videoStream, err := fuseFile.getStream(readRequest.Pid)
	if err != nil {
		message := fmt.Sprintf("Failed to get video stream for pid %d", readRequest.Pid)
		fuseFile.logger.Error(message, err)
		return err
	}

	// TODO buffer pool
	buffer := make([]byte, readRequest.Size)

	bytesRead, err := videoStream.ReadAt(buffer, uint64(readRequest.Offset))
	if err != nil {
		message := fmt.Sprintf("Failed to read video stream for pid %d", readRequest.Pid)
		fuseFile.logger.Error(message, err)
		return err
	}

	readResponse.Data = buffer[:bytesRead]

	return nil
}

var _ fs.HandleFlusher = &File{}

func (fuseFile *File) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	videoStream, ok := fuseFile.streams.Load(flushRequest.Pid)
	if ok {
		videoStream.Close()
	}

	return nil
}

func (fuseFile *File) getStream(pid uint32) (*stream.Stream, error) {
	existingStream, ok := fuseFile.streams.Load(pid)
	if ok {
		if !existingStream.IsClosed() {
			return existingStream, nil
		}

		fuseFile.streams.Delete(pid)
	}

	response, err := fuseFile.client.GetVideoUrl(context.Background(), &vfs_api.GetVideoUrlRequest{
		Identifier: fuseFile.identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to get video url for pid %d", pid)
		fuseFile.logger.Error(message, err)
		return nil, err
	}

	newStream := stream.NewStream(response.Url, fuseFile.size)

	fuseFile.streams.Store(pid, newStream)

	return newStream, nil
}
