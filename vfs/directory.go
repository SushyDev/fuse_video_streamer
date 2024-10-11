package vfs

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type Directory struct {
	fileSystem *FileSystem
}

func (directory *Directory) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("Setting root directory attributes")

	a.Inode = 1
	a.Mode = os.ModeDir

	return nil
}

func (directory *Directory) Lookup(ctx context.Context, name string) (fs.Node, error) {
	directory.fileSystem.mu.Lock()
	defer directory.fileSystem.mu.Unlock()

	fileNode, exists := directory.fileSystem.files[name]
	if !exists {
		return nil, syscall.ENOENT
	}

	fmt.Println("Found file", name)

	return fileNode, nil
}

func (directory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	directory.fileSystem.mu.Lock()
	defer directory.fileSystem.mu.Unlock()

	var entries []fuse.Dirent
	inode := uint64(2)

	for name := range directory.fileSystem.files {
		entries = append(entries, fuse.Dirent{
			Name:  name,
			Type:  fuse.DT_File,
			Inode: inode,
		})
		inode++
	}

	return entries, nil
}

func (directory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	directory.fileSystem.mu.Lock()
	defer directory.fileSystem.mu.Unlock()

	if removeRequest.Dir {
		return syscall.ENOSYS
	}

	fmt.Println("Removing file", removeRequest.Name)

	file, exists := directory.fileSystem.files[removeRequest.Name]
	if !exists {
		fmt.Println("File does not exist")
		return syscall.ENOENT
	}

	return file.Remove(ctx)
}
