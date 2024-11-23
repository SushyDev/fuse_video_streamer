package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"
)

var _ fs.Node = &File{}
var _ fs.NodeOpener = &File{}
var _ fs.Handle = &File{}
var _ fs.HandleReader = &File{}
var _ fs.HandleWriter = &File{}
var _ fs.HandleReleaser = &File{}

type File struct {
	file   *vfs.File
	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewFile(file *vfs.File) *File {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &File{
		file:   file,
		logger: fuseLogger,
	}
}

func (node *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.file != nil {
		attr.Size = node.file.GetSize()
		attr.Inode = node.file.GetIdentifier()
		attr.Mode = os.ModePerm | 0o777
	}

	attr.Atime = time.Unix(0, 0)
	attr.Mtime = time.Unix(0, 0)
	attr.Ctime = time.Unix(0, 0)

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (node *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	node.logger.Infof("Opening file %s - %d", node.file.GetName(), node.file.GetIdentifier())

	openResponse.Flags |= fuse.OpenKeepCache

	return node, nil
}

func (node *File) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	if node.file != nil {
		node.logger.Infof("Releasing file %s", node.file.GetName())

		node.file.Close()
	}

	return nil
}

func (node *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.file == nil {
		fmt.Println("File is nil")
		readResponse.Data = []byte("This file was created to verify if '/Users/sushy/Documents/Projects/fuse_video_steamer/mnt' is writable. It should've been automatically deleted. Feel free to delete it.")
		return nil
	}

	if readRequest.Dir {
		return fmt.Errorf("read request is for a directory")
	}

	buffer := make([]byte, readRequest.Size)
	bytesRead, err := node.file.Read(buffer, readRequest.Offset, readRequest.Pid)
	if err != nil {
		return fmt.Errorf("failed to read from file: %w", err)
	}

	readResponse.Data = buffer[:bytesRead]

	return nil
}

func (node *File) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	// TODO SONARR SUPPORT
	writeResponse.Size = len(writeRequest.Data)

	return nil
}
