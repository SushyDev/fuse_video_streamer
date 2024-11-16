package vfs

import (
	"sync/atomic"
)

type VirtualFileSystem struct {
	root      *Directory
	IDCounter atomic.Uint64

	index *Index

	// mu sync.RWMutex TODO
}

func NewVirtualFileSystem() *VirtualFileSystem {
	index := newIndex()

	fileSystem := &VirtualFileSystem{
		index: index,
	}

	fileSystem.root = fileSystem.NewDirectory(nil, "root")

	return fileSystem
}

func (fileSystem *VirtualFileSystem) GetRoot() *Directory {
	return fileSystem.root
}

// --- Directory

func (fileSystem *VirtualFileSystem) NewDirectory(parent *Directory, name string) *Directory {
	id := fileSystem.IDCounter.Add(1)

	index := newIndex()

	directory := &Directory{
		identifier: id,
		name:       name,
		parent:     parent,
		index:      index,
	}

	fileSystem.index.registerDirectory(directory)

	if parent != nil {
		parent.index.registerDirectory(directory)
	}

	return directory
}

func (fileSystem *VirtualFileSystem) RemoveDirectory(directory *Directory) {
	directory.index.close()

	if directory.parent != nil {
		directory.parent.index.deregisterDirectory(directory)
	}

	fileSystem.index.deregisterDirectory(directory)
}

func (fileSystem *VirtualFileSystem) RenameDirectory(directory *Directory, name string, parent *Directory) *Directory {
	if directory.parent != nil {
		directory.parent.index.deregisterDirectory(directory)
	}

	directory.name = name
	directory.parent = parent

	if parent != nil {
		parent.index.registerDirectory(directory)
	}

	return directory
}

func (fileSystem *VirtualFileSystem) GetDirectory(ID uint64) *Directory {
	return fileSystem.index.getDirectory(ID)
}

func (fileSystem *VirtualFileSystem) FindDirectory(name string) *Directory {
	return fileSystem.index.findDirectory(name)
}

func (fileSystem *VirtualFileSystem) ListDirectories() []*Directory {
	return fileSystem.index.listDirectories()
}

// --- File

func (fileSystem *VirtualFileSystem) NewFile(parent *Directory, name string, videoUrl string, fetchUrl string, size uint64) *File {
	ID := fileSystem.IDCounter.Add(1)

	file := &File{
		identifier: ID,
		name:       name,
		videoUrl:   videoUrl,
		fetchUrl:   fetchUrl,
		size:       size,
		parent:     parent,
	}

	fileSystem.index.registerFile(file)

	if parent != nil {
		parent.index.registerFile(file)
	}

	return file
}

func (fileSystem *VirtualFileSystem) RemoveFile(file *File) {
	if file.parent != nil {
		file.parent.index.deregisterFile(file)
	}

	fileSystem.index.deregisterFile(file)
}

func (fileSystem *VirtualFileSystem) RenameFile(file *File, name string, parent *Directory) *File {
	if file.parent != nil {
		file.parent.index.deregisterFile(file)
	}

	file.name = name
	file.parent = parent

	if parent != nil {
		parent.index.registerFile(file)
	}

	return file
}

func (fileSystem *VirtualFileSystem) GetFile(ID uint64) *File {
	return fileSystem.index.getFile(ID)
}

func (fileSystem *VirtualFileSystem) FindFile(name string) *File {
	return fileSystem.index.findFile(name)
}
