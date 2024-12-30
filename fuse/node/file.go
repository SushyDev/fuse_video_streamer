package node

import (
	"context"
	"fmt"
	"io"
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

	videoStreams sync.Map
	size         uint64

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
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	openResponse.Flags |= fuse.OpenKeepCache

	return fuseFile, nil
}

var _ fs.HandleReader = &File{}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	videoStream, err := fuseFile.getVideoStream(readRequest.Pid)
	if err != nil {
		message := fmt.Sprintf("Failed to get video stream for pid %d", readRequest.Pid)
		fuseFile.logger.Error(message, err)
		return err
	}

	_, err = videoStream.Seek(uint64(readRequest.Offset), io.SeekStart)
	if err != nil {
		message := fmt.Sprintf("Failed to seek video stream for pid %d", readRequest.Pid)
		fuseFile.logger.Error(message, err)
		return err
	}

	buffer := make([]byte, readRequest.Size)

	bytesRead, err := videoStream.Read(buffer)
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

	videoStream, ok := fuseFile.videoStreams.Load(flushRequest.Pid)
	if ok {
		videoStream.(*stream.Stream).Close()
	}

	return nil
}

// --- Helpers

func (fuseFile *File) getVideoStream(pid uint32) (*stream.Stream, error) {
	videoStream, ok := fuseFile.videoStreams.Load(pid)
	if ok {
		return videoStream.(*stream.Stream), nil
	}

	response, err := fuseFile.client.GetVideoUrl(context.Background(), &vfs_api.GetVideoUrlRequest{
		Identifier: fuseFile.identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to get video url for pid %d", pid)
		fuseFile.logger.Error(message, err)
		return nil, err
	}

	newVideoStream := stream.NewStream(response.Url, fuseFile.size)

	fuseFile.videoStreams.Store(pid, newVideoStream)

	return newVideoStream, nil
}
