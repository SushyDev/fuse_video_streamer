package node

import (
	"context"
	"fmt"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"
	vfs_node "fuse_video_steamer/vfs/node"
	"log"
	"os"
	"sync"
	"syscall"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

var _ fs.Handle = &Directory{}

type Directory struct {
	vfs        *vfs.FileSystem
	identifier uint64

	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewDirectory(vfs *vfs.FileSystem, identifier uint64) *Directory {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &Directory{
		vfs:        vfs,
		identifier: identifier,
		logger:     fuseLogger,
	}
}

var _ fs.Node = &Directory{}

func (fuseDirectory *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	attr.Inode = fuseDirectory.identifier
	attr.Mode = os.ModeDir | 0o777
	attr.Valid = 1

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	attr.Atime = attr.Ctime
	attr.Mtime = attr.Ctime
	attr.Crtime = attr.Ctime

	return nil
}

var _ fs.NodeOpener = &Directory{}

func (fuseDirectory *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	return fuseDirectory, nil
}

// Todo Inode matching
var _ fs.NodeRequestLookuper = &Directory{}

func (fuseDirectory *Directory) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		log.Printf("Lookup: Failed to get directory\n%v", err)
		return nil, err
	}

	childNode, err := fuseDirectory.vfs.GetChildNode(lookupRequest.Name, directory)
	if err != nil {
		log.Printf("Lookup: Failed to get child node\n%v", err)
		return nil, err
	}

	if childNode == nil {
		return nil, syscall.ENOENT
	}

	switch childNode.GetType() {
	case vfs_node.FileNode:
		identifier := childNode.GetIdentifier()
		return NewFile(fuseDirectory.vfs, identifier), nil
	case vfs_node.DirectoryNode:
		identifier := childNode.GetIdentifier()
		return NewDirectory(fuseDirectory.vfs, identifier), nil
	}

	return nil, syscall.ENOENT
}

var _ fs.HandleReadDirAller = &Directory{}

func (fuseDirectory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		log.Printf("Failed to get directory\n%v", err)
		return nil, err
	}


	childNodes, err := fuseDirectory.vfs.GetChildNodes(directory)
	if err != nil {
		log.Printf("Failed to get child nodes\n%v", err)
		return nil, err
	}

	var entries []fuse.Dirent
	for _, childNode := range childNodes {
		switch childNode.GetType() {
		case vfs_node.FileNode:
			entries = append(entries, fuse.Dirent{
				Name:  childNode.GetName(),
				Type:  fuse.DT_File,
				Inode: childNode.GetIdentifier(),
			})
		case vfs_node.DirectoryNode:
			entries = append(entries, fuse.Dirent{
				Name:  childNode.GetName(),
				Type:  fuse.DT_Dir,
				Inode: childNode.GetIdentifier(),
			})
		}
	}

	return entries, nil
}

var _ fs.NodeRemover = &Directory{}

func (fuseDirectory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
        log.Printf("Failed to get directory\n%v", err)
		return err
	}

	childNode, err := fuseDirectory.vfs.GetChildNode(removeRequest.Name, directory)
	if err != nil {
        log.Printf("Failed to get child node\n%v", err)
		return err
	}

	if childNode == nil {
        log.Printf("Failed to get child node\n%v", syscall.ENOENT)
		return syscall.ENOENT
	}

	switch childNode.GetType() {
	case vfs_node.FileNode:
		file, err := fuseDirectory.vfs.GetFile(childNode.GetIdentifier())
		if err != nil {
            log.Printf("Failed to get file\n%v", err)
			return err
		}

		err = fuseDirectory.vfs.DeleteFile(file)
		if err != nil {
            log.Printf("Failed to delete file\n%v", err)
			return err
		}
	case vfs_node.DirectoryNode:
		directory, err := fuseDirectory.vfs.GetDirectory(childNode.GetIdentifier())
		if err != nil {
            log.Printf("Failed to get directory\n%v", err)
			return err
		}

		err = fuseDirectory.vfs.DeleteDirectory(directory)
		if err != nil {
            log.Printf("Failed to delete directory\n%v", err)
			return err
		}
	}

	return nil
}

var _ fs.NodeRenamer = &Directory{}

func (fuseDirectory *Directory) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
        log.Printf("Failed to get directory\n%v", err)
		return err
	}

	newParent, err := newNode.(*Directory).getDirectory()
	if err != nil {
        log.Printf("Failed to get new parent\n%v", err)
		return err
	}

	childNode, err := fuseDirectory.vfs.GetChildNode(request.OldName, directory)
	if err != nil {
        log.Printf("Failed to get child node\n%v", err)
		return err
	}

	if childNode == nil {
        log.Printf("Failed to get child node\n%v", syscall.ENOENT)
		return syscall.ENOENT
	}

	switch childNode.GetType() {
	case vfs_node.FileNode:
		file, err := fuseDirectory.vfs.GetFile(childNode.GetIdentifier())
		if err != nil {
            log.Printf("Failed to get file\n%v", err)
			return err
		}

		_, err = fuseDirectory.vfs.UpdateFile(file, request.NewName, newParent, file.GetSize(), file.GetHost())
		if err != nil {
            log.Printf("Failed to update file\n%v", err)
			return err
		}
	case vfs_node.DirectoryNode:
		directory, err := fuseDirectory.vfs.GetDirectory(childNode.GetIdentifier())
		if err != nil {
            log.Printf("Failed to get directory\n%v", err)
			return err
		}

		_, err = fuseDirectory.vfs.UpdateDirectory(directory, request.NewName, newParent)
		if err != nil {
            log.Printf("Failed to update directory\n%v", err)
			return err
		}
	}

	return nil
}

var _ fs.NodeCreater = &Directory{}

func (fuseDirectory *Directory) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
        log.Printf("Failed to get directory\n%v", err)
		return nil, nil, err
	}

	newFile, err := fuseDirectory.vfs.CreateFile(request.Name, directory, 0, "")
	if err != nil {
        log.Printf("Failed to create file\n%v", err)
		return nil, nil, err
	}

	newFileNode := NewFile(fuseDirectory.vfs, newFile.GetNode().GetIdentifier())

	return fuseDirectory, newFileNode, nil
}

var _ fs.NodeMkdirer = &Directory{}

func (fuseDirectory *Directory) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
        log.Printf("Failed to get directory\n%v", err)
		return nil, err
	}

	newDirectory, err := fuseDirectory.vfs.CreateDirectory(request.Name, directory)
	if err != nil {
        log.Printf("Failed to create directory\n%v", err)
		return nil, err
	}

	newDirectoryNode := NewDirectory(fuseDirectory.vfs, newDirectory.GetNode().GetIdentifier())

	return newDirectoryNode, nil
}

var _ fs.NodeLinker = &Directory{}

func (fuseDirectory *Directory) Link(ctx context.Context, request *fuse.LinkRequest, node fs.Node) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
        log.Printf("Failed to get directory\n%v", err)
		return nil, err
	}

    node, ok := node.(*File)
    if !ok {
        log.Printf("Failed to get file\n%v", syscall.EINVAL)
        return nil, syscall.EINVAL
    }

    file, err := node.(*File).getFile()
    if err != nil {
        log.Printf("Failed to get file\n%v", err)
        return nil, err
    }


    fmt.Println("Linking file", file.GetNode().GetName())

    newFile, err := fuseDirectory.vfs.UpdateFile(file, request.NewName, directory, file.GetSize(), file.GetHost())
    if err != nil {
        log.Printf("Failed to update file\n%v", err)
        return nil, err
    }

    newFileNode := NewFile(fuseDirectory.vfs, newFile.GetNode().GetIdentifier())

	return newFileNode, nil
}

// --- Helpers

func (fuseDirectory *Directory) getDirectory() (*vfs_node.Directory, error) {
	directory, err := fuseDirectory.vfs.GetDirectory(fuseDirectory.identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get directory\n%w", err)
	}

	if directory == nil {
		return nil, fmt.Errorf("failed to get directory\n%w", syscall.ENOENT)
	}

	return directory, nil
}
