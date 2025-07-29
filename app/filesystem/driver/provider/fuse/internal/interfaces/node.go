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
	New(Tree) (RootNodeService, error)
}

type RootNodeService interface {
	useClosable

	New() (RootNode, error)
}

type RootNode interface {
	useClosable
	useIdentifier

	Node

	fs.NodeOpener
	fs.NodeRequestLookuper
}

// --- Directory

type DirectoryNodeServiceFactory interface {
	New(interfaces_filesystem_client.Client, Tree) (DirectoryNodeService, error)
}

type DirectoryNodeService interface {
	useClosable

	New(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (DirectoryNode, error)
}

type DirectoryNode interface {
	useClosable
	useIdentifier
	useRemoteIdentifier

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
	New(interfaces_filesystem_client.Client, Tree) (StreamableNodeService, error)
}

type StreamableNodeService interface {
	useClosable

	New(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (StreamableNode, error)
}

type StreamableNode interface {
	useClosable
	useIdentifier
	useRemoteIdentifier

	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() interfaces_filesystem_client.Client
}

// --- File

type FileNodeServiceFactory interface {
	New(interfaces_filesystem_client.Client, Tree) (FileNodeService, error)
}

type FileNodeService interface {
	useClosable

	New(parentDirectoryNode DirectoryNode, remoteIdentifier uint64) (FileNode, error)
}

type FileNode interface {
	useClosable
	useIdentifier
	useRemoteIdentifier

	Node

	fs.NodeOpener

	GetSize() uint64
	GetClient() interfaces_filesystem_client.Client
}
