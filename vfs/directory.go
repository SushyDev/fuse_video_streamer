package vfs

type Directory struct {
	identifier uint64
	name       string

	parent *Directory
	index  *Index

	fileSystem *FileSystem

	// mu sync.RWMutex TODO
}

func (directory *Directory) GetIdentifier() uint64 {
	return directory.identifier
}

func (directory *Directory) GetName() string {
	return directory.name
}

func (directory *Directory) GetParent() *Directory {
	return directory.parent
}

func (directory *Directory) GetFileSystem() *FileSystem {
	return directory.fileSystem
}

// --- Directory

func (directory *Directory) FindDirectory(name string) *Directory {
	return directory.index.findDirectory(name)
}

func (directory *Directory) ListDirectories() []*Directory {
	return directory.index.listDirectories()
}

func (directory *Directory) RemoveDirectory(entry *Directory) {
	directory.index.deregisterDirectory(entry)
}

// --- File

func (directory *Directory) FindFile(name string) *File {
	return directory.index.findFile(name)
}

func (directory *Directory) ListFiles() []*File {
	return directory.index.listFiles()
}
