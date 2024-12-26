package node

import (
	"context"
	"io"
	"os"
	"sync"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

var _ fs.Handle = &File{}
var _ fs.HandleReleaser = &File{}

type File struct {
	client     vfs_api.FileSystemServiceClient
	identifier uint64

	videoStreams sync.Map
	size         uint64

	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewFile(client vfs_api.FileSystemServiceClient, identifier uint64, size uint64) *File {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &File{
		client:     client,
		identifier: identifier,

		size: size,

		logger: fuseLogger,
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
		return err
	}

	_, err = videoStream.Seek(uint64(readRequest.Offset), io.SeekStart)
	if err != nil {
		return err
	}

	buffer := make([]byte, readRequest.Size)

	bytesRead, err := videoStream.Read(buffer)
	if err != nil {
		return err
	}

	readResponse.Data = buffer[:bytesRead]

	return nil
}

var _ fs.HandleReleaser = &File{}

func (fuseFile *File) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	videoStream, ok := fuseFile.videoStreams.Load(releaseRequest.Pid)
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
		return nil, err
	}

	newVideoStream := stream.NewStream(response.Url, fuseFile.size)

	fuseFile.videoStreams.Store(pid, newVideoStream)

	return newVideoStream, nil
}
