package registry

import (
	"context"
	"sync"

	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
)

type Registry struct {
	nodes []interfaces.Node

	ctx    context.Context
	cancel context.CancelFunc
}

var instance *Registry

func GetInstance() *Registry {
	if instance != nil {
		return instance
	}

	instance = &Registry{
		nodes: []interfaces.Node{},
	}

	return instance
}

func (registry *Registry) Add(node interfaces.Node) {
	registry.nodes = append(registry.nodes, node)
}

func (registry *Registry) CloseNodes() {
	var wg sync.WaitGroup

	for _, node := range registry.nodes {
		wg.Add(1)

		go func() {
			defer wg.Done()
			node.Close()
			node = nil
		}()
	}

	registry.nodes = nil

	wg.Wait()
}
