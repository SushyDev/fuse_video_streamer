package fuse

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"

	"debrid_drive/logger"
	"debrid_drive/vfs"
)

var _ fs.Node = &FileNode{}
var _ fs.Handle = &FileNode{}
var _ fs.HandleReader = &FileNode{}
var _ fs.HandleReleaser = &FileNode{}
var _ fs.NodeRemover = &FileNode{}

type FileNode struct {
	file *vfs.File

	mu sync.RWMutex
}

func NewFileNode(file *vfs.File) *FileNode {
	return &FileNode{
		file: file,
	}
}

func (node *FileNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	attr.Size = node.file.Size
	attr.Inode = node.file.ID
	attr.Mode = os.ModePerm

	attr.Atime = time.Unix(0, 0)
	attr.Mtime = time.Unix(0, 0)
	attr.Ctime = time.Unix(0, 0)

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (node *FileNode) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	logger.Logger.Infof("Removing file %s", node.file.Name)

	return nil
}

func (node *FileNode) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	logger.Logger.Infof("Opening file %s - %d", node.file.Name, node.file.Size)

	openResponse.Flags |= fuse.OpenKeepCache

	return node, nil
}

func (node *FileNode) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	logger.Logger.Infof("Releasing file %s", node.file.Name)

	node.file.Close()

	return nil
}

func (node *FileNode) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	// fmt.Printf("Reading %d bytes at offset %d\n", readRequest.Size, readRequest.Offset)

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
