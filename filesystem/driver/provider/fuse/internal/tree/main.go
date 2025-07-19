package tree

import (
	"fmt"
	"syscall"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type Tree struct {
	increment uint64

	nodes map[uint64]interfaces_fuse.Node

}

var _ interfaces_fuse.Tree = &Tree{}

func New() *Tree {
	return &Tree{
		increment: 0,
		nodes:     make(map[uint64]interfaces_fuse.Node),
	}
}

func (t *Tree) GetNextIdentifier() uint64 {
	t.increment++
	return t.increment
}

func (t *Tree) RegisterNodeOnIdentifier(identifier uint64, node interfaces_fuse.Node) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if _, exists := t.nodes[identifier]; exists {
		return syscall.EEXIST
	}

	t.nodes[identifier] = node

	return nil
}
