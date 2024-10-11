package vfs

import (
	"fmt"
	"sync"

	"bazil.org/fuse/fs"
)

type FileSystem struct {
	files map[string]*File
	mu    sync.Mutex
}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	fmt.Println("Returning root directory")

	return &Directory{
		fileSystem: fileSystem,
	}, nil
}
