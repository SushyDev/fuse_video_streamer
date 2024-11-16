package vfs

import ()

type Index struct {
	directories map[uint64]*Directory
	files       map[uint64]*File
}

func newIndex() *Index {
	return &Index{
		directories: make(map[uint64]*Directory),
		files:       make(map[uint64]*File),
	}
}

func (index *Index) close() {
	for _, directory := range index.directories {
		index.deregisterDirectory(directory)
	}

	for _, file := range index.files {
		index.deregisterFile(file)
	}
}

// --- Directory

func (index *Index) registerDirectory(directory *Directory) {
	index.directories[directory.identifier] = directory
}

func (index *Index) deregisterDirectory(directory *Directory) {
	delete(index.directories, directory.identifier)
}

func (index *Index) getDirectory(ID uint64) *Directory {
	return index.directories[ID]
}

func (index *Index) findDirectory(name string) *Directory {
	for _, directory := range index.directories {
		if directory.name == name {
			return directory
		}
	}

	return nil
}

func (index *Index) listDirectories() []*Directory {
	directories := make([]*Directory, 0, len(index.directories))

	for _, directory := range index.directories {
		directories = append(directories, directory)
	}

	return directories
}

// --- File

func (index *Index) registerFile(file *File) {
	index.files[file.identifier] = file
}

func (index *Index) deregisterFile(file *File) {
	delete(index.files, file.identifier)
}

func (index *Index) getFile(ID uint64) *File {
	return index.files[ID]
}

func (index *Index) findFile(name string) *File {
	for _, file := range index.files {
		if file.name == name {
			return file
		}
	}

	return nil
}

func (index *Index) listFiles() []*File {
	files := make([]*File, 0, len(index.files))

	for _, file := range index.files {
		files = append(files, file)
	}

	return files
}
