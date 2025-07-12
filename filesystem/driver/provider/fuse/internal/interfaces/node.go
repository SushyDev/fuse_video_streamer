package interfaces

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"

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
	New(filesystem_client_interfaces.Client) (DirectoryNodeService, error)
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

	GetClient() filesystem_client_interfaces.Client
}

// --- Streamable

type StreamableNodeServiceFactory interface {
	New(filesystem_client_interfaces.Client) (StreamableNodeService, error)
}

type StreamableNodeService interface {
	useClosable

	New(identifier uint64) (StreamableNode, error)
}

type StreamableNode interface {
	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() filesystem_client_interfaces.Client
}

// --- File

type FileNodeServiceFactory interface {
	New(filesystem_client_interfaces.Client) (FileNodeService, error)
}

type FileNodeService interface {
	useClosable

	New(identifier uint64) (FileNode, error)
}

type FileNode interface {
	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() filesystem_client_interfaces.Client
}
	
