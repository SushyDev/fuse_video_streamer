package tree

import (
	"fmt"
	"sync"
	"syscall"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

type Tree struct {
	increment uint64
	mutex     sync.RWMutex
	nodes     map[uint64]interfaces_node.AbstractNode
}

var _ interfaces_node.Tree = &Tree{}

func New() *Tree {
	return &Tree{
		increment: 0,
		nodes:     make(map[uint64]interfaces_node.AbstractNode),
	}
}

func (t *Tree) GetNextIdentifier() uint64 {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.increment++
	return t.increment
}

func (t *Tree) RegisterNode(node interfaces_node.AbstractNode) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	identifier := node.GetIdentifier()
	if _, exists := t.nodes[identifier]; exists {
		return syscall.EEXIST
	}

	t.nodes[identifier] = node

	return nil
}
