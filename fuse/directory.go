package fuse

import (
	"context"
	"debrid_drive/logger"
	"os"
	"sync"
	"syscall"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Directory struct {
	name        string
	iNode       uint64
	fileSystem  *FileSystem
	directories map[string]*Directory
	files       map[string]*File
	mu          sync.RWMutex
}

func (directory *Directory) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = directory.iNode
	a.Mode = os.ModeDir
	a.Valid = 1

	return nil
}

func (directory *Directory) Lookup(ctx context.Context, name string) (fs.Node, error) {
	directory.mu.RLock()
	defer directory.mu.RUnlock()

	file, fileExists := directory.files[name]
	if fileExists && file != nil {
		return file, nil
	}

	directory, directoryExists := directory.directories[name]
	if directoryExists && directory != nil {
		return directory, nil
	}

	return nil, syscall.ENOENT
}

func (directory *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	// openResponse.Flags |= fuse.OpenDirectIO

	return directory, nil
}

func (directory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	directory.mu.RLock()
	defer directory.mu.RUnlock()

	var entries []fuse.Dirent

	for _, file := range directory.files {
		entries = append(entries, fuse.Dirent{
			Name:  file.name,
			Type:  fuse.DT_File,
			Inode: file.iNode,
		})
	}

	for _, directory := range directory.directories {
		entries = append(entries, fuse.Dirent{
			Name:  directory.name,
			Type:  fuse.DT_Dir,
			Inode: directory.iNode,
		})
	}

	return entries, nil
}

func (directory *Directory) ReadDir(ctx context.Context) ([]fuse.Dirent, error) {
	return directory.ReadDirAll(ctx)
}

func (directory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	directory.mu.Lock()
	defer directory.mu.Unlock()

	logger.Logger.Infof("Remove request: %v", removeRequest)

	if removeRequest.Dir {
		return syscall.ENOSYS
	}

	file, exists := directory.files[removeRequest.Name]
	if !exists {
		logger.Logger.Warnf("File %s does not exist", removeRequest.Name)
		return syscall.ENOENT
	}

	if err := file.Remove(ctx); err != nil {
		return err
	}

	delete(directory.files, removeRequest.Name)

	logger.Logger.Infof("Removed file %s", removeRequest.Name)

	return nil
}
