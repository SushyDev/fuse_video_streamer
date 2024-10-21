package vfs

import (
	"fmt"
	"sync"
)

type Directory struct {
	ID          uint64
	Name        string
	Directories map[uint64]*Directory
	Files       map[uint64]*File
	Parent      *Directory

	mu sync.RWMutex

	fileSystem *FileSystem
}

func NewDirectory(fileSystem *FileSystem, parent *Directory, name string) (*Directory, error) {
	if fileSystem == nil {
		return nil, fmt.Errorf("file system is nil")
	}

	ID := fileSystem.IDCounter.Add(1)

	directory := &Directory{
		ID:          ID,
		Name:        name,
		Directories: make(map[uint64]*Directory),
		Files:       make(map[uint64]*File),
		Parent:      parent,

		fileSystem: fileSystem,
	}

	return directory, nil
}

func (directory *Directory) Rename(name string) {
	directory.mu.Lock()
	defer directory.mu.Unlock()

	directory.Name = name
}

func (directory *Directory) GetSubDirectory(name string) *Directory {
	for _, directory := range directory.Directories {
		if directory.Name == name {
			return directory
		}
	}

	return nil
}

func (directory *Directory) AddSubDirectory(name string) (*Directory, error) {
	foundDirectory := directory.GetSubDirectory(name)
	if foundDirectory != nil {
		return nil, fmt.Errorf("directory already exists")
	}

	newDirectory, err := NewDirectory(directory.fileSystem, directory, name)
	if err != nil {
		return nil, err
	}

	directory.registerSubDirectory(newDirectory)

	directory.fileSystem.RegisterDirectory(newDirectory)

	return newDirectory, nil
}

func (directory *Directory) registerSubDirectory(newDirectory *Directory) {
	directory.mu.Lock()
	defer directory.mu.Unlock()

	directory.Directories[newDirectory.ID] = newDirectory
}

func (directory *Directory) GetFile(name string) *File {
	for _, file := range directory.Files {
		if file.Name == name {
			return file
		}
	}

	return nil
}

func (directory *Directory) AddFile(name string, videoUrl string, fetchUrl string, size uint64) (*File, error) {
	foundFile := directory.GetFile(name)
	if foundFile != nil {
		return nil, fmt.Errorf("file already exists")
	}

	newFile, err := NewFile(directory.fileSystem, directory, name, videoUrl, fetchUrl, size)
	if err != nil {
		return nil, err
	}

	directory.mu.Lock()
	directory.Files[newFile.ID] = newFile
	directory.mu.Unlock()

	return newFile, nil
}
