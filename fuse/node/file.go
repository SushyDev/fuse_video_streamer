package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"
	vfs_node "fuse_video_steamer/vfs/node"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

var _ fs.Handle = &File{}
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

var _ fs.Node = &File{}

func (fuseFile *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		fuseFile.logger.Infof("Attr: Failed to get file: %v", err)
		return err
	}

	attr.Inode = fuseFile.identifier
	attr.Mode = os.ModePerm | 0o777
	attr.Size = vfsFile.GetSize()

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

var _ fs.NodeOpener = &File{}

func (fuseFile *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	openResponse.Flags |= fuse.OpenKeepCache

	return fuseFile, nil
}

var _ fs.HandleReader = &File{}

func (fuseFile *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		fuseFile.logger.Infof("Read: Failed to get file: %v", err)
		readResponse.Data = []byte("This file was created to verify if '/Users/sushy/Documents/Projects/fuse_video_steamer/mnt' is writable. It should've been automatically deleted. Feel free to delete it.")
		return nil
	}

	if readRequest.Dir {
		fuseFile.logger.Infof("Read: Read request is for a directory")
		return fmt.Errorf("read request is for a directory")
	}

	if vfsFile.GetVideoURL() != "" {
		buffer := make([]byte, readRequest.Size)
		bytesRead, err := vfsFile.Read(buffer, readRequest.Offset, readRequest.Pid)
		if err != nil {
			fuseFile.logger.Infof("Read: Failed to read file: %v", err)
			return err
		}

		readResponse.Data = buffer[:bytesRead]
	}

	return nil
}

var _ fs.HandleWriter = &File{}

func (fuseFile *File) Write(ctx context.Context, writeRequest *fuse.WriteRequest, writeResponse *fuse.WriteResponse) error {
	fuseFile.mu.RLock()
	defer fuseFile.mu.RUnlock()

	// TODO SONARR SUPPORT
	writeResponse.Size = len(writeRequest.Data)

	return nil
}

var _ fs.HandleReleaser = &File{}

func (fuseFile *File) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	fuseFile.mu.Lock()
	defer fuseFile.mu.Unlock()

	vfsFile, err := fuseFile.getFile()
	if err != nil {
		fuseFile.logger.Infof("Release: Failed to get file: %v", err)
		return err
	}

	fuseFile.logger.Infof("Releasing file %s", vfsFile.GetNode().GetName())

	vfsFile.Close()

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
