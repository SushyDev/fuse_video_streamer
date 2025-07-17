package interfaces

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"

	"github.com/anacrolix/fuse/fs"
)

// --- Generic

type Node interface {
	useClosable

	fs.Node

	GetIdentifier() uint64
}

// --- Root

type RootNodeServiceFactory interface {
	New() (RootNodeService, error)
}

type RootNodeService interface {
	useClosable

	New() (RootNode, error)
}

type RootNode interface {
	useClosable

	Node

	fs.NodeOpener
	fs.NodeRequestLookuper
}

// --- Directory

type DirectoryNodeServiceFactory interface {
	New(interfaces_filesystem_client.Client) (DirectoryNodeService, error)
}

type DirectoryNodeService interface {
	useClosable

	New(identifier uint64) (DirectoryNode, error)
}

type DirectoryNode interface {
	Node

	fs.NodeOpener
	fs.NodeRequestLookuper
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeLinker

	GetClient() interfaces_filesystem_client.Client
}

// --- Streamable

type StreamableNodeServiceFactory interface {
	New(interfaces_filesystem_client.Client) (StreamableNodeService, error)
}

type StreamableNodeService interface {
	useClosable

	New(identifier uint64) (StreamableNode, error)
}

type StreamableNode interface {
	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() interfaces_filesystem_client.Client
}

// --- File

type FileNodeServiceFactory interface {
	New(interfaces_filesystem_client.Client) (FileNodeService, error)
}

type FileNodeService interface {
	useClosable

	New(identifier uint64) (FileNode, error)
}

type FileNode interface {
	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() interfaces_filesystem_client.Client
}
