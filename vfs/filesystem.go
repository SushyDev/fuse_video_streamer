package vfs

import (
	"fmt"
	"strings"
	"sync"

	"bazil.org/fuse/fs"
	"debrid_drive/logger"
)

type FileSystem struct {
	mu           sync.RWMutex
	iNodeCounter uint64
	directory    *Directory
}

func NewFileSystem() *FileSystem {
	fileSystem := &FileSystem{
		iNodeCounter: 1,
	}

	fileSystem.mu.Lock()
	fileSystem.directory = fileSystem.NewDirectory("root")
	fileSystem.mu.Unlock()

	return fileSystem
}

func (fileSystem *FileSystem) NewDirectory(name string) *Directory {
	fileSystem.iNodeCounter += 1

	return &Directory{
		name:        name,
		iNode:       fileSystem.iNodeCounter,
		directories: make(map[string]*Directory),
		files:       make(map[string]*File),
		fileSystem:  fileSystem,
	}
}

func (fileSystem *FileSystem) NewFile(name string, videoUrl string, videoSize int64) *File {
	fileSystem.iNodeCounter += 1

	return &File{
		name:     name,
		iNode:    fileSystem.iNodeCounter,
		videoUrl: videoUrl,
		chunks:   0,
		size:     videoSize,
	}
}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	return fileSystem.directory, nil
}

func (fileSystem *FileSystem) AddDirectory(parent *Directory, name string) (*Directory, error) {
	if parent == nil {
		return nil, fmt.Errorf("parent directory is nil")
	}

	directory := fileSystem.NewDirectory(name)

	fileSystem.mu.Lock()
	parent.directories[name] = directory
	fileSystem.mu.Unlock()

	logger.Logger.Infof("Created directory %s with inode %d", directory.name, directory.iNode)

	return directory, nil
}

func (fileSystem *FileSystem) RemoveDirectory(parent *Directory, name string) {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	delete(parent.directories, name)
}

func (fileSystem *FileSystem) FindDirectory(path string) (*Directory, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	if path == "" || path == "/" {
		return fileSystem.directory, nil
	}

	components := strings.Split(path, "/")
	currentDirectory := fileSystem.directory

	for _, name := range components {
		newDirectory, exists := currentDirectory.directories[name]

		if !exists {
			return nil, fmt.Errorf("directory %s does not exist", name)
		}

		currentDirectory = newDirectory
	}

	return currentDirectory, nil
}

func (fileSystem *FileSystem) AddFile(parent *Directory, name string, videoUrl string, videoSize int64) (*File, error) {
	if parent == nil {
		return nil, fmt.Errorf("parent directory is nil")
	}

	file := fileSystem.NewFile(name, videoUrl, videoSize)

	fileSystem.mu.Lock()
	parent.files[name] = file
	defer fileSystem.mu.Unlock()

	logger.Logger.Infof("Created file %s with inode %d to directory %s", name, file.iNode, parent.name)

	return file, nil
}

func (fileSystem *FileSystem) RemoveFile(parent *Directory, name string) {
	fileSystem.mu.Lock()
	defer fileSystem.mu.Unlock()

	delete(parent.files, name)
}
