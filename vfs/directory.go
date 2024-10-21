package vfs

import (
	"fmt"
)

type Directory struct {
	ID          uint64
	Name        string
	Directories map[uint64]*Directory
	Files       map[uint64]*File
	Parent      *Directory

	// mu sync.RWMutex TODO

	fileSystem *VirtualFileSystem
}

func NewDirectory(fileSystem *VirtualFileSystem, parent *Directory, name string) (*Directory, error) {
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
	directory.Name = name
}

// Get direct child
func (directory *Directory) GetDirectory(name string) *Directory {
	for _, directory := range directory.Directories {
		if directory.Name == name {
			return directory
		}
	}

	return nil
}

// Add direct child
func (directory *Directory) AddDirectory(name string) (*Directory, error) {
	foundDirectory := directory.GetDirectory(name)
	if foundDirectory != nil {
		return nil, fmt.Errorf("directory already exists")
	}

	newDirectory, err := NewDirectory(directory.fileSystem, directory, name)
	if err != nil {
		return nil, err
	}

	directory.registerDirectory(newDirectory)

	directory.fileSystem.RegisterDirectory(newDirectory)

	return newDirectory, nil
}

func (directory *Directory) RemoveDirectory(name string) error {
	foundDirectory := directory.GetDirectory(name)
	if foundDirectory == nil {
		return fmt.Errorf("directory not found")
	}

	directory.deregisterDirectory(foundDirectory.ID)

	directory.fileSystem.DeleteDirectory(foundDirectory.ID)

	return nil
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

	directory.registerFile(newFile)

	directory.fileSystem.RegisterFile(newFile)

	return newFile, nil
}

func (directory *Directory) RemoveFile(name string) error {
	foundFile := directory.GetFile(name)
	if foundFile == nil {
		return fmt.Errorf("file not found")
	}

	directory.deregisterFile(foundFile.ID)

	directory.fileSystem.DeregisterFile(foundFile.ID)

	return nil
}

// --- Helpers

func (directory *Directory) registerDirectory(newDirectory *Directory) {
	directory.Directories[newDirectory.ID] = newDirectory
}

func (directory *Directory) deregisterDirectory(ID uint64) {
	delete(directory.Directories, ID)
}

func (directory *Directory) registerFile(newFile *File) {
	directory.Files[newFile.ID] = newFile
}

func (directory *Directory) deregisterFile(ID uint64) {
	delete(directory.Files, ID)
}
