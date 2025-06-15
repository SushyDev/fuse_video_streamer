package interfaces

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"

	"github.com/anacrolix/fuse/fs"
)

// --- Generic

type Node interface {
	fs.Node

	GetIdentifier() uint64
	Close() error
}

// --- Root

type RootNodeServiceFactory interface {
	New() (RootNodeService, error)
}

type RootNodeService interface {
	New() (RootNode, error)
	Close() error
}

type RootNode interface {
	Node

	fs.NodeOpener
	fs.NodeRequestLookuper
}

// --- Directory

type DirectoryNodeServiceFactory interface {
	New(filesystem_client_interfaces.Client) (DirectoryNodeService, error)
}

type DirectoryNodeService interface {
	New(identifier uint64) (DirectoryNode, error)
	Close() error
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
}

// --- Streamable

type StreamableNodeServiceFactory interface {
	New(filesystem_client_interfaces.Client) (StreamableNodeService, error)
}

type StreamableNodeService interface {
	New(identifier uint64) (StreamableNode, error)
	Close() error
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
	New(identifier uint64) (FileNode, error)
}

type FileNode interface {
	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() filesystem_client_interfaces.Client
}
	
