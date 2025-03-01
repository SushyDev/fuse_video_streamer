package registry

import (
	"context"
	"sync"

	"fuse_video_steamer/fuse/node"
)

type Registry struct {
	files map[uint64]*node.File
	directories map[uint64]*node.Directory

	ctx context.Context
	cancel context.CancelFunc
}

var instance *Registry

func GetInstance() *Registry {
	if instance == nil {
		instance = &Registry{
			files: make(map[uint64]*node.File),
			directories: make(map[uint64]*node.Directory),
		}

		return instance
	}

	return instance
}

func (registry *Registry) AddFile(identifier uint64, file *node.File) {
	registry.files[identifier] = file
}

func (registry *Registry) AddDirectory(identifier uint64, directory *node.Directory) {
	registry.directories[identifier] = directory
}

func (registry *Registry) CloseNodes() {
	var wg sync.WaitGroup

	for _, file := range registry.files {
		wg.Add(1)

		go func() {
			defer wg.Done()
			file.Close()
		}()
	}

	for _, directory := range registry.directories {
		wg.Add(1)

		go func() {
			defer wg.Done()
			directory.Close()
		}()
	}

	wg.Wait()
}
