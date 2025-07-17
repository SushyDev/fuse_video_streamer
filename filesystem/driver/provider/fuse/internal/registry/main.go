package registry

// Todo registry for client instead of this?

import (
	"context"
	"sync"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type Registry struct {
	nodes []interfaces_fuse.Node

	ctx    context.Context
	cancel context.CancelFunc
}

var instances = map[string]*Registry{}

func GetInstance(client interfaces_filesystem_client.Client) *Registry {
	if client == nil {
		return nil
	}

	fileSystem := client.GetFileSystem()

	root, err := fileSystem.Root(client.GetName())
	if err != nil {
		return nil
	}

	if instance, ok := instances[root.GetName()]; ok {
		return instance
	}

	ctx, cancel := context.WithCancel(context.Background())

	instance := &Registry{
		nodes:  []interfaces_fuse.Node{},
		ctx:    ctx,
		cancel: cancel,
	}

	instances[root.GetName()] = instance

	return instance
}

func (registry *Registry) Add(node interfaces_fuse.Node) {
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
}
