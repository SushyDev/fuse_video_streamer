package fuse

import (
	"context"
	"debrid_drive/vfs"
	"fmt"
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
var _ fs.NodeCreater = &DirectoryNode{}
var _ fs.NodeMkdirer = &DirectoryNode{}
var _ fs.NodeLinker = &DirectoryNode{}

type DirectoryNode struct {
	directory  *vfs.Directory
	fileSystem *FuseFileSystem

	mu sync.RWMutex
}

func (node *DirectoryNode) Attr(ctx context.Context, attr *fuse.Attr) error {
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

func (node *DirectoryNode) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	return node, nil
}

// Todo Inode matching
func (node *DirectoryNode) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	file := node.directory.FindFile(lookupRequest.Name)
	if file != nil {
		return NewFileNode(file), nil
	}

	directory := node.directory.FindDirectory(lookupRequest.Name)
	if directory != nil {
		return node.fileSystem.NewDirectoryNode(directory), nil
	}

	return nil, syscall.ENOENT
}

func (node *DirectoryNode) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
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

func (node *DirectoryNode) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fmt.Println("Remove request on directory", removeRequest.Name)

	return nil // TODO SONARR SUPPORT

	node.mu.Lock()

	if removeRequest.Dir {
		entry := node.directory.FindDirectory(removeRequest.Name)

		node.fileSystem.VirtualFileSystem.RemoveDirectory(entry)
	} else {
		entry := node.directory.FindFile(removeRequest.Name)

		node.fileSystem.VirtualFileSystem.RemoveFile(entry)
	}

	node.mu.Unlock()

	_, err := node.ReadDirAll(ctx)
	if err != nil {
		fuseLogger.Errorf("Failed to read directory %s: %v", node.directory.GetName(), err)
		return err
	}

	return nil
}

func (node *DirectoryNode) Rename(ctx context.Context, request *fuse.RenameRequest, newNode fs.Node) error {
	fuseLogger.Infof("Rename request on directory %s: %v", node.directory.GetName())

	node.mu.Lock()

	oldDirectory := node.directory.FindDirectory(request.OldName)
	oldFile := node.directory.FindFile(request.OldName)

	if oldDirectory == nil && oldFile == nil {
		return syscall.ENOENT
	}

	newParentDirectory := newNode.(*DirectoryNode).directory

	if oldDirectory != nil {
		fmt.Println("Rename directory", request.NewName)

		node.fileSystem.VirtualFileSystem.RenameDirectory(oldDirectory, request.NewName, newParentDirectory)
	}

	if oldFile != nil {
		fmt.Println("Rename file", request.NewName, "parent:", newParentDirectory.GetName())

		node.fileSystem.VirtualFileSystem.RenameFile(oldFile, request.NewName, newParentDirectory)
	}

	node.mu.Unlock()

	_, err := node.ReadDirAll(ctx)
	if err != nil {
		fuseLogger.Errorf("Failed to read directory %s: %v", node.directory.GetName(), err)
		return err
	}

	return nil
}

func (node *DirectoryNode) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fmt.Println("Create request on directory", node.directory.GetName(), "new:", request.Name)

	file := node.fileSystem.VirtualFileSystem.NewFile(node.directory, request.Name, "", "", 0)

	// node.fileSystem.InvalidateEntry(node.directory.GetIdentifier(), request.Name)

	fileNode := NewFileNode(file)

	return fileNode, fileNode, nil
}

func (node *DirectoryNode) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fmt.Println("Mkdir request on directory", node.directory.GetName(), "new:", request.Name)

	directory := node.fileSystem.VirtualFileSystem.NewDirectory(node.directory, request.Name)

	// node.fileSystem.InvalidateEntry(node.directory.GetIdentifier(), request.Name)

	return node.fileSystem.NewDirectoryNode(directory), nil
}

func (node *DirectoryNode) Link(ctx context.Context, request *fuse.LinkRequest, old fs.Node) (fs.Node, error) {
	oldFile := old.(*FileNode).file

	newFile := node.fileSystem.VirtualFileSystem.NewFile(node.directory, request.NewName, oldFile.GetVideoUrl(), oldFile.GetFetchUrl(), oldFile.GetSize())

	// node.fileSystem.InvalidateEntry(node.directory.GetIdentifier(), request.NewName)

	return NewFileNode(newFile), nil
}
