package registry

// Todo registry for client instead of this?

import (
	"context"
	"sync"

	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	client_interfaces "fuse_video_steamer/filesystem/client/interfaces"
)

type Registry struct {
	nodes []interfaces.Node

	ctx    context.Context
	cancel context.CancelFunc
}

var instances = map[string]*Registry{}

func GetInstance(client client_interfaces.Client) *Registry {
	if client == nil {
		return nil
	}

	fileSystem := client.GetFileSystem()

	root, err := fileSystem.Root(client.GetName())
	if err != nil {
		panic(err)
	}

	if instance, ok := instances[root.GetName()]; ok {
		return instance
	}

	ctx, cancel := context.WithCancel(context.Background())

	instance := &Registry{
		nodes: []interfaces.Node{},
		ctx:   ctx,
		cancel: cancel,
	}

	instances[root.GetName()] = instance

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

func Close() {
	for _, instance := range instances {
		instance.CloseNodes()
		instance = nil
	}

	instances = nil
}
