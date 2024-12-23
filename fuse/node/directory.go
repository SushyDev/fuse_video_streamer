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

var _ fs.Node = &Directory{}
var _ fs.Handle = &Directory{}
var _ fs.NodeOpener = &Directory{}
var _ fs.NodeRequestLookuper = &Directory{}
var _ fs.HandleReadDirAller = &Directory{}
var _ fs.NodeRemover = &Directory{}
var _ fs.NodeRenamer = &Directory{}
var _ fs.NodeCreater = &Directory{}
var _ fs.NodeMkdirer = &Directory{}
var _ fs.NodeLinker = &Directory{}

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

func (fuseDirectory *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
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

func (fuseDirectory *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	return fuseDirectory, nil
}

// Todo Inode matching
func (fuseDirectory *Directory) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		log.Printf("Lookup: Failed to get directory\n%v", err)
		return nil, err
	}

	childFile, err := fuseDirectory.vfs.FindFile(lookupRequest.Name, directory)
	if err != nil {
		log.Printf("Lookup: Failed to find file\n%v", err)
		return nil, err
	}

	if childFile != nil {
		identifier := childFile.GetNode().GetIdentifier()
		return NewFile(fuseDirectory.vfs, identifier), nil
	}

	childDirectory, err := fuseDirectory.vfs.FindDirectory(lookupRequest.Name, directory)
	if err != nil {
		log.Printf("Lookup: Failed to find directory\n%v", err)
		return nil, err
	}

	if childDirectory != nil {
		identifier := childDirectory.GetNode().GetIdentifier()
		return NewDirectory(fuseDirectory.vfs, identifier), nil
	}

	return nil, syscall.ENOENT
}

func (fuseDirectory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		log.Printf("Failed to get directory\n%v", err)
		return nil, syscall.ENOENT
	}

	var entries []fuse.Dirent

	childNodes, err := fuseDirectory.vfs.GetChildNodes(directory)
	if err != nil {
		log.Print(err)
		return nil, err
	}

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

func (fuseDirectory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fmt.Println("Remove request on directory", removeRequest.Name)

	return nil // TODO SONARR SUPPORT

	fuseDirectory.mu.Lock()

	// if removeRequest.Dir {
	// 	entry := node.directory.FindDirectoryByName(removeRequest.Name)
	// } else {
	// 	entry := node.directory.FindFileByName(removeRequest.Name)
	// }

	fuseDirectory.mu.Unlock()

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		return err
	}

	_, err = fuseDirectory.ReadDirAll(ctx)
	if err != nil {
		fuseDirectory.logger.Errorf("Failed to read directory %s\n%v", directory.GetNode().GetName(), err)
		return err
	}

	return nil
}

func (fuseDirectory *Directory) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	fmt.Println("Rename request on entry", request.OldName, "new:", request.NewName)
	return nil

	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		return err
	}

	fuseDirectory.logger.Infof("Rename request on directory %s\n%v", directory.GetNode().GetName())

	fuseDirectory.mu.Lock()

	// oldFile := fuseDirectory.vfs.FindNodeByName(request.OldName)
	// if oldFile == nil {
	// 	return syscall.ENOENT
	// }
	//
	// newDirectory, err := newNode.(*Directory).getDirectory()
	// if err != nil {
	// 	return err
	// }
	//
	// oldFile.Move(newDirectory, request.NewName)

	fuseDirectory.mu.Unlock()

	return nil
}

func (fuseDirectory *Directory) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		return nil, nil, err
	}

	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

	newFile, err := fuseDirectory.vfs.CreateFile(request.Name, directory, 0, "")
	if err != nil {
		return nil, nil, err
	}

	newFileNode := NewFile(fuseDirectory.vfs, newFile.GetNode().GetIdentifier())

	return fuseDirectory, newFileNode, nil
}

func (fuseDirectory *Directory) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	directory, err := fuseDirectory.getDirectory()
	if err != nil {
		return nil, err
	}

	fmt.Println("Mkdir request on directory", directory.GetNode().GetName(), "new:", request.Name)

	fuseDirectory.mu.Lock()

	newDirectory, err := fuseDirectory.vfs.CreateDirectory(request.Name, directory)
	if err != nil {
		return nil, err
	}

	fuseDirectory.mu.Unlock()

	newDirectoryNode := NewDirectory(fuseDirectory.vfs, newDirectory.GetNode().GetIdentifier())

	return newDirectoryNode, nil
}

func (fuseDirectory *Directory) Link(ctx context.Context, request *fuse.LinkRequest, node fs.Node) (fs.Node, error) {
	//    fileSystemDirectory, err := fuseDirectory.getDirectory()
	//    if err != nil {
	//        return nil, err
	//    }
	//
	// fmt.Println("Link request on directory", fileSystemDirectory, "new:", request.NewName)
	//
	//    fuseDirectory.mu.Lock()
	//
	// oldNode, ok := node.(*File)
	//    if !ok {
	//        return nil, syscall.EINVAL
	//    }
	//
	//    // todo
	//    // get generic node entry from vfs
	//    // call link on generic entry
	//
	//    newNode := oldNode.Link(fileSystemDirectory, request.NewName)
	//
	//    fuseDirectory.mu.Unlock()
	//
	// return NewFile(newNode), nil
	return nil, nil
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
