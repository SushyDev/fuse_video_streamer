package fuse

import (
	"context"
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
}

func NewDirectoryNode(directory *vfs.Directory) *DirectoryNode {
	return &DirectoryNode{
		directory: directory,
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
			node := NewDirectoryNode(directory)

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

func (node *DirectoryNode) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	node.mu.Lock()

    if removeRequest.Dir {
        err := node.directory.RemoveDirectory(removeRequest.Name)
        if err != nil {
            fuseLogger.Errorf("Failed to remove directory %s: %v", removeRequest.Name, err)
            return err
        }
    } else {
        err := node.directory.RemoveFile(removeRequest.Name)
        if err != nil {
            fuseLogger.Errorf("Failed to remove file %s: %v", removeRequest.Name, err)
            return err
        }
    }

    node.mu.Unlock()

    _, err := node.ReadDirAll(ctx)
    if err != nil {
        fuseLogger.Errorf("Failed to read directory %s: %v", node.directory.Name, err)
        return err
    }

	return nil
}

func (node *DirectoryNode) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	node.mu.Lock()

	fuseLogger.Infof("Rename request on directory %s: %v", node.directory.Name, request)

	directory := node.directory.GetDirectory(request.OldName)
	file := node.directory.GetFile(request.OldName)

	switch {
	case directory != nil:
		directory.Rename(request.NewName)
		break
	case file != nil:
		file.Rename(request.NewName)
		break
	default:
		return syscall.ENOENT
	}

    node.mu.Unlock()

	_, err := node.ReadDirAll(ctx)
	if err != nil {
        fuseLogger.Errorf("Failed to read directory %s: %v", node.directory.Name, err)
		return err
	}

	return nil
}
