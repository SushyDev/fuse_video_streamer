package abstract

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

type Node struct {
	client           interfaces_filesystem_client.Client
	identifier       uint64
	remoteIdentifier uint64
}

var _ interfaces_node.AbstractNode = &Node{}

func NewNode(
	client interfaces_filesystem_client.Client,
	identifier uint64,
	remoteIdentifier uint64,
) *Node {
	return &Node{
		client:           client,
		identifier:       identifier,
		remoteIdentifier: remoteIdentifier,
	}
}

func (node *Node) GetClient() interfaces_filesystem_client.Client {
	return node.client
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetRemoteIdentifier() uint64 {
	return node.remoteIdentifier
}

func (node *Node) Close() error {
	// No resources to release in this base implementation.
	// Override in derived types if necessary.
	return nil
}

func (node *Node) IsClosed() bool {
	// No state to track in this base implementation.
	// Override in derived types if necessary.
	return false
}
