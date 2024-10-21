package fuse

import (
	"context"
	"debrid_drive/logger"
	"debrid_drive/vfs"
	"os"
	"sync"
	"syscall"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.Node = &DirectoryNode{}
var _ fs.Handle = &DirectoryNode{}
var _ fs.NodeOpener = &DirectoryNode{}
var _ fs.NodeRequestLookuper = &DirectoryNode{}
var _ fs.HandleReadDirAller = &DirectoryNode{}
var _ fs.NodeRemover = &DirectoryNode{}
var _ fs.NodeRenamer = &DirectoryNode{}

type DirectoryNode struct {
	directory *vfs.Directory

	mu sync.RWMutex

	connection *fuse.Conn
}

func NewDirectoryNode(connection *fuse.Conn, directory *vfs.Directory) *DirectoryNode {
	return &DirectoryNode{
		directory: directory,

		connection: connection,
	}
}

func (node *DirectoryNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = node.directory.ID
	attr.Mode = os.ModeDir | 0775
	attr.Valid = 1

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	attr.Atime = attr.Ctime
	attr.Mtime = attr.Ctime
	attr.Crtime = attr.Ctime

	return nil
}

func (node *DirectoryNode) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	return node, nil
}

// Todo Inode matching
func (node *DirectoryNode) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	for _, file := range node.directory.Files {
		if file.Name == lookupRequest.Name {
			node := NewFileNode(file)

			return node, nil
		}
	}

	for _, directory := range node.directory.Directories {
		if directory.Name == lookupRequest.Name {
			node := NewDirectoryNode(node.connection, directory)

			return node, nil
		}
	}

	return nil, syscall.ENOENT
}

func (node *DirectoryNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	var entries []fuse.Dirent

	for _, file := range node.directory.Files {
		entries = append(entries, fuse.Dirent{
			Name:  file.Name,
			Type:  fuse.DT_File,
			Inode: file.ID,
		})
	}

	for _, directory := range node.directory.Directories {
		entries = append(entries, fuse.Dirent{
			Name:  directory.Name,
			Type:  fuse.DT_Dir,
			Inode: directory.ID,
		})
	}

	return entries, nil
}

// func (node *DirectoryNode) ReadDir(ctx context.Context) ([]fuse.Dirent, error) {
// 	return node.ReadDirAll(ctx)
// }

func (node *DirectoryNode) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	logger.Logger.Infof("Remove request: %v", removeRequest)

	// if removeRequest.Dir {
	// 	return syscall.ENOSYS
	// }
	//
	// file, exists := directory.files[removeRequest.Name]
	// if !exists {
	// 	logger.Logger.Warnf("File %s does not exist", removeRequest.Name)
	// 	return syscall.ENOENT
	// }
	//
	// if err := file.Remove(ctx, removeRequest); err != nil {
	// 	return err
	// }
	//
	// delete(directory.files, removeRequest.Name)
	//
	// logger.Logger.Infof("Removed file %s", removeRequest.Name)

	return nil
}

// func (node *DirectoryNode) Invalidate() error {
// 	node.mu.Lock()
// 	defer node.mu.Unlock()
//
// 	return node.fileSystem.connection.InvalidateEntry(node.directory.ID, node.directory.Name)
// }

func (node *DirectoryNode) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	node.directory.Rename(request.NewName)

	_, err := node.ReadDirAll(ctx)
	if err != nil {
		return err
	}

	return nil
}
