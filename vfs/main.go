package vfs

import (
	"fmt"
	"sync/atomic"
)

type VirtualFileSystem struct {
	Root         *Directory
	DirectoryMap map[uint64]*Directory
	FileMap      map[uint64]*File

	IDCounter atomic.Uint64

	// mu sync.RWMutex TODO
}

func NewVirtualFileSystem() *VirtualFileSystem {
	fileSystem := &VirtualFileSystem{
		DirectoryMap: make(map[uint64]*Directory),
		FileMap:      make(map[uint64]*File),
	}

	fileSystem.CreateRootDirectory()

	return fileSystem
}

func (fileSystem *VirtualFileSystem) CreateRootDirectory() {
	directory, _ := NewDirectory(fileSystem, nil, "root")

	fileSystem.DirectoryMap[directory.ID] = directory

	fileSystem.Root = directory
}

func (fileSystem *VirtualFileSystem) GetDirectory(ID uint64) (*Directory, error) {
	for directoryID, directory := range fileSystem.DirectoryMap {
		if directoryID == ID {
			return directory, nil
		}
	}

	return nil, fmt.Errorf("Directory with ID %d not found", ID)
}

func (fileSystem *VirtualFileSystem) GetFile(ID uint64) (*File, error) {
	for fileID, file := range fileSystem.FileMap {
		if fileID == ID {
			return file, nil
		}
	}

	return nil, fmt.Errorf("File with ID %d not found", ID)
}

func (fileSystem *VirtualFileSystem) RegisterDirectory(directory *Directory) {
	fileSystem.DirectoryMap[directory.ID] = directory
}

func (fileSystem *VirtualFileSystem) DeleteDirectory(ID uint64) {
	delete(fileSystem.DirectoryMap, ID)
}

func (fileSystem *VirtualFileSystem) RegisterFile(file *File) {
	fileSystem.FileMap[file.ID] = file
}

func (fileSystem *VirtualFileSystem) DeregisterFile(ID uint64) {
	delete(fileSystem.FileMap, ID)
}
