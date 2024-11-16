package vfs

type Directory struct {
	identifier uint64
	name       string

	parent *Directory
	index  *Index

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

// --- Directory

func (directory *Directory) FindDirectory(name string) *Directory {
	return directory.index.findDirectory(name)
}

func (directory *Directory) ListDirectories() []*Directory {
	return directory.index.listDirectories()
}

// --- File

func (directory *Directory) NewFile(entry *File) *File {
	entry.parent = directory

	directory.index.registerFile(entry)

	return entry
}

func (directory *Directory) RemoveFile(entry *File) {
	directory.index.deregisterFile(entry)
}

func (directory *Directory) FindFile(name string) *File {
	return directory.index.findFile(name)
}

func (directory *Directory) ListFiles() []*File {
	return directory.index.listFiles()
}

// --- Helpers
