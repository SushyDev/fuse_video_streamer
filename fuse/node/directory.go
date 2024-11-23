package node

import (
	"context"
	"fmt"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"
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
	directory *vfs.Directory

	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewDirectory(directory *vfs.Directory) *Directory {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &Directory{
		directory: directory,
		logger:    fuseLogger,
	}
}

func (node *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = node.directory.GetIdentifier()
	attr.Mode = os.ModeDir | 0o777
	attr.Valid = 1

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	attr.Atime = attr.Ctime
	attr.Mtime = attr.Ctime
	attr.Crtime = attr.Ctime

	return nil
}

func (node *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	return node, nil
}

// Todo Inode matching
func (node *Directory) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	file := node.directory.FindFile(lookupRequest.Name)
	if file != nil {
		return NewFile(file), nil
	}

	directory := node.directory.FindDirectory(lookupRequest.Name)
	if directory != nil {
		return NewDirectory(directory), nil
	}

	return nil, syscall.ENOENT
}

func (node *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	var entries []fuse.Dirent

	for _, file := range node.directory.ListFiles() {
		entries = append(entries, fuse.Dirent{
			Name:  file.GetName(),
			Type:  fuse.DT_File,
			Inode: file.GetIdentifier(),
		})
	}

	for _, directory := range node.directory.ListDirectories() {
		entries = append(entries, fuse.Dirent{
			Name:  directory.GetName(),
			Type:  fuse.DT_Dir,
			Inode: directory.GetIdentifier(),
		})
	}

	return entries, nil
}

func (node *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fmt.Println("Remove request on directory", removeRequest.Name)

	return nil // TODO SONARR SUPPORT

	node.mu.Lock()

	fileSystem := node.directory.GetFileSystem()

	if removeRequest.Dir {
		entry := node.directory.FindDirectory(removeRequest.Name)

		fileSystem.RemoveDirectory(entry)
	} else {
		entry := node.directory.FindFile(removeRequest.Name)

		fileSystem.RemoveFile(entry)
	}

	node.mu.Unlock()

	_, err := node.ReadDirAll(ctx)
	if err != nil {
		node.logger.Errorf("Failed to read directory %s: %v", node.directory.GetName(), err)
		return err
	}

	return nil
}

func (node *Directory) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	node.logger.Infof("Rename request on directory %s: %v", node.directory.GetName())

	node.mu.Lock()

	oldDirectory := node.directory.FindDirectory(request.OldName)
	oldFile := node.directory.FindFile(request.OldName)

	if oldDirectory == nil && oldFile == nil {
		return syscall.ENOENT
	}

	newParentDirectory := newNode.(*Directory).directory

	fileSystem := node.directory.GetFileSystem()

	if oldDirectory != nil {
		fileSystem.RenameDirectory(oldDirectory, request.NewName, newParentDirectory)
	}

	if oldFile != nil {
		fileSystem.RenameFile(oldFile, request.NewName, newParentDirectory)
	}

	node.mu.Unlock()

	// _, err := node.ReadDirAll(ctx)
	// if err != nil {
	// 	node.logger.Errorf("Failed to read directory %s: %v", node.directory.GetName(), err)
	// 	return err
	// }

	return nil
}

func (node *Directory) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fmt.Println("Create request on directory", node.directory.GetName(), "new:", request.Name)

	fileSystem := node.directory.GetFileSystem()

	file := fileSystem.NewFile(node.directory, request.Name, "", "", 0)

	// node.fileSystem.InvalidateEntry(node.directory.GetIdentifier(), request.Name)

	fileNode := NewFile(file)

	return fileNode, fileNode, nil
}

func (node *Directory) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fmt.Println("Mkdir request on directory", node.directory.GetName(), "new:", request.Name)

	fileSystem := node.directory.GetFileSystem()

	directory := fileSystem.NewDirectory(node.directory, request.Name)

	// node.fileSystem.InvalidateEntry(node.directory.GetIdentifier(), request.Name)

	return NewDirectory(directory), nil
}

func (node *Directory) Link(ctx context.Context, request *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
	fmt.Println("Link request on directory", node.directory.GetName(), "new:", request.NewName)

	oldFile := old.(*File).file

	fileSystem := node.directory.GetFileSystem()

	newFile := fileSystem.RenameFile(oldFile, request.NewName, node.directory)

	return NewFile(newFile), nil
}
