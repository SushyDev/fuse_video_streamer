package registry

import (
	"context"
	"sync"
	"time"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

// Registry is the injectable interface for node lifecycle tracking.
type Registry interface {
	Add(interfaces_node.AbstractNode)
	CloseNodes()
}

type registry struct {
	mu     sync.RWMutex
	nodes  []interfaces_node.AbstractNode
	ctx    context.Context
	cancel context.CancelFunc
}

var _ Registry = &registry{}

// New creates and returns a new Registry instance.
func New() Registry {
	ctx, cancel := context.WithCancel(context.Background())

	return &registry{
		nodes:  make([]interfaces_node.AbstractNode, 0),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (registry *registry) Add(node interfaces_node.AbstractNode) {
	if node == nil {
		return
	}

	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.nodes = append(registry.nodes, node)
}

func (registry *registry) CloseNodes() {
	registry.mu.RLock()
	nodes := make([]interfaces_node.AbstractNode, len(registry.nodes))
	copy(nodes, registry.nodes)
	registry.mu.RUnlock()

	var wg sync.WaitGroup
	done := make(chan struct{})

	for _, node := range nodes {
		wg.Add(1)

		go func(nodeInstance interfaces_node.AbstractNode) {
			defer wg.Done()
			nodeInstance.Close()
		}(node)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for nodes to close with a 30 second timeout
	select {
	case <-done:
		// All nodes closed successfully
	case <-time.After(30 * time.Second):
		// Timeout - some nodes failed to close, but continue anyway
		// In a production system, this would be logged
	}
}
