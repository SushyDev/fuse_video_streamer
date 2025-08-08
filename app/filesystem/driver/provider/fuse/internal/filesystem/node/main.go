package node

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"github.com/anacrolix/fuse/fs"
)

// Trait

type Tree interface {
	GetNextIdentifier() uint64
	RegisterNode(node AbstractNode) error
}

// Node

type AbstractNode interface {
	interfaces_fuse.UseClient
	interfaces_fuse.UseClosable
	interfaces_fuse.UseIdentifier
	interfaces_fuse.UseRemoteIdentifier
}

type RootNode interface {
	interfaces_fuse.UseClosable

	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
}

type DirectoryNode interface {
	AbstractNode

	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeLinker
}

type FileNode interface {
	AbstractNode

	interfaces_fuse.UseSize

	fs.Node
	fs.NodeOpener
}

type StreamableNode interface {
	AbstractNode

	interfaces_fuse.UseSize

	fs.Node
	fs.NodeOpener
}

// Service

type RootNodeService interface {
	interfaces_fuse.UseClosable

	NewNode() (RootNode, error)
}

type DirectoryNodeService interface {
	interfaces_fuse.UseClosable

	NewNode(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (DirectoryNode, error)
}

type FileNodeService interface {
	interfaces_fuse.UseClosable

	NewNode(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (FileNode, error)
}

type StreamableNodeService interface {
	interfaces_fuse.UseClosable

	NewNode(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (StreamableNode, error)
}

// Factory

type RootNodeServiceFactory interface {
	NewService(Tree) (RootNodeService, error)
}

type DirectoryNodeServiceFactory interface {
	NewService(interfaces_filesystem_client.Client, Tree) (DirectoryNodeService, error)
}

type StreamableNodeServiceFactory interface {
	NewService(interfaces_filesystem_client.Client, Tree) (StreamableNodeService, error)
}

type FileNodeServiceFactory interface {
	NewService(interfaces_filesystem_client.Client, Tree) (FileNodeService, error)
}
