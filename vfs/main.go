package vfs

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type FileSystem struct {
	Root         *Directory
	DirectoryMap map[uint64]*Directory
	FileMap      map[uint64]*File // TODO

	IDCounter atomic.Uint64

	mu sync.RWMutex
}

func NewFileSystem() *FileSystem {
	fileSystem := &FileSystem{
		DirectoryMap: make(map[uint64]*Directory),
	}

	fileSystem.CreateRootDirectory()

	return fileSystem
}

func (fileSystem *FileSystem) CreateRootDirectory() {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	directory, _ := NewDirectory(fileSystem, nil, "root")

	fileSystem.DirectoryMap[directory.ID] = directory

	fileSystem.Root = directory
}

func (fileSystem *FileSystem) GetDirectory(ID uint64) (*Directory, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	for directoryID, directory := range fileSystem.DirectoryMap {
		if directoryID == ID {
			return directory, nil
		}
	}

	return nil, fmt.Errorf("Directory with ID %d not found", ID)
}

func (fileSystem *FileSystem) RegisterDirectory(directory *Directory) {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	fileSystem.DirectoryMap[directory.ID] = directory
}
