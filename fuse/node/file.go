package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"
	vfs_node "fuse_video_steamer/vfs/node"
)

var _ fs.Node = &File{}
var _ fs.NodeOpener = &File{}
var _ fs.Handle = &File{}
var _ fs.HandleReader = &File{}
var _ fs.HandleWriter = &File{}
var _ fs.HandleReleaser = &File{}

type File struct {
	vfs        *vfs.FileSystem
	identifier uint64

	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewFile(vfs *vfs.FileSystem, identifier uint64) *File {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &File{
		vfs:        vfs,
		identifier: identifier,
		logger:     fuseLogger,
	}
}

func (fuseFile *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		return err
	}

	attr.Size = vfsFile.GetSize()
	attr.Inode = vfsFile.GetNode().GetIdentifier()
	attr.Mode = os.ModePerm | 0o777

	attr.Atime = time.Unix(0, 0)
	attr.Mtime = time.Unix(0, 0)
	attr.Ctime = time.Unix(0, 0)

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (fuseFile *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	virtualFileSystemFile, err := fuseFile.getFile()
	if err != nil {
		return nil, err
	}

	fuseFile.logger.Infof("Opening file %s - %d", virtualFileSystemFile.GetNode().GetName(), virtualFileSystemFile.GetNode().GetIdentifier())

	openResponse.Flags |= fuse.OpenKeepCache

	return fuseFile, nil
}

func (fuseFile *File) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		return err
	}

	fuseFile.logger.Infof("Releasing file %s", vfsFile.GetNode().GetName())

	vfsFile.Close()

	return nil
}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		fmt.Println("File is nil")
		readResponse.Data = []byte("This file was created to verify if '/Users/sushy/Documents/Projects/fuse_video_steamer/mnt' is writable. It should've been automatically deleted. Feel free to delete it.")
		return nil
	}

	if readRequest.Dir {
		return fmt.Errorf("read request is for a directory")
	}

	if vfsFile.GetVideoURL() != "" {
		buffer := make([]byte, readRequest.Size)
		bytesRead, err := vfsFile.Read(buffer, readRequest.Offset, readRequest.Pid)
		if err != nil {
			return fmt.Errorf("failed to read from file: %w", err)
		}

		readResponse.Data = buffer[:bytesRead]
	}

	return nil
}

func (fuseFile *File) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	// TODO SONARR SUPPORT
	writeResponse.Size = len(writeRequest.Data)

	return nil
}

// --- Helpers

// TODO: Call only once in constructor
func (fuseFile *File) getFile() (*vfs_node.File, error) {
	vfsFile, err := fuseFile.vfs.GetFile(fuseFile.identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if vfsFile == nil {
		return nil, fmt.Errorf("failed to get file: %w", syscall.ENOENT)
	}

	return vfsFile, nil
}
