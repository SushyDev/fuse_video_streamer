package registry

import (
	"context"
	"sync"

	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/vfs_api"
)

type Registry struct {
	nodes []interfaces.Node

	ctx    context.Context
	cancel context.CancelFunc
}

var instances = map[string]*Registry{}

func GetInstance(client vfs_api.FileSystemServiceClient) *Registry {
	if client == nil {
		return nil
	}

	response, err := client.Root(context.Background(), &vfs_api.RootRequest{})
	if err != nil {
		panic(err)
	}

	if instance, ok := instances[response.Root.Name]; ok {
		return instance
	}

	ctx, cancel := context.WithCancel(context.Background())

	instance := &Registry{
		nodes: []interfaces.Node{},
		ctx:   ctx,
		cancel: cancel,
	}

	instances[response.Root.Name] = instance

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
