package vfs

import (
	"sync/atomic"
)

type FileSystem struct {
	root      *Directory
	IDCounter atomic.Uint64

	index *Index

	// mu sync.RWMutex TODO
}

func NewFileSystem() *FileSystem {
	index := newIndex()

	fileSystem := &FileSystem{
		index: index,
	}

	fileSystem.root = fileSystem.NewDirectory(nil, "root")

	return fileSystem
}

func (fileSystem *FileSystem) GetRoot() *Directory {
	return fileSystem.root
}

// --- Directory

func (fileSystem *FileSystem) NewDirectory(parent *Directory, name string) *Directory {
	id := fileSystem.IDCounter.Add(1)

	index := newIndex()

	directory := &Directory{
		identifier: id,
		name:       name,
		parent:     parent,
		index:      index,

		fileSystem: fileSystem,
	}

	fileSystem.index.registerDirectory(directory)

	if parent != nil {
		parent.index.registerDirectory(directory)
	}

	return directory
}

func (fileSystem *FileSystem) RemoveDirectory(directory *Directory) {
	directory.index.close()

	if directory.parent != nil {
		directory.parent.index.deregisterDirectory(directory)
	}

	fileSystem.index.deregisterDirectory(directory)
}

func (fileSystem *FileSystem) RenameDirectory(directory *Directory, name string, parent *Directory) *Directory {
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

func (fileSystem *FileSystem) GetDirectory(ID uint64) *Directory {
	return fileSystem.index.getDirectory(ID)
}

func (fileSystem *FileSystem) FindDirectory(name string) *Directory {
	return fileSystem.index.findDirectory(name)
}

func (fileSystem *FileSystem) ListDirectories() []*Directory {
	return fileSystem.index.listDirectories()
}

// --- File

func (fileSystem *FileSystem) NewFile(parent *Directory, name string, host string, fetchUrl string, size uint64) *File {
	ID := fileSystem.IDCounter.Add(1)

	file := &File{
		identifier: ID,
		name:       name,
		videoUrl:   host,
		host:       fetchUrl,
		size:       size,
		parent:     parent,

		fileSystem: fileSystem,
	}

	fileSystem.index.registerFile(file)

	if parent != nil {
		parent.index.registerFile(file)
	}

	// event := &Event{
	//     EventType: EventFileCreated,
	//     File: file,
	// }
	//
	// http.NewRequest("GET", fetchUrl, event)

	return file
}

func (fileSystem *FileSystem) RemoveFile(file *File) {
	if file.parent != nil {
		file.parent.index.deregisterFile(file)
	}

	fileSystem.index.deregisterFile(file)
}

func (fileSystem *FileSystem) RenameFile(file *File, name string, parent *Directory) *File {
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

func (fileSystem *FileSystem) GetFile(ID uint64) *File {
	return fileSystem.index.getFile(ID)
}

func (fileSystem *FileSystem) FindFile(name string) *File {
	return fileSystem.index.findFile(name)
}
