package interfaces

import (
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse/fs"
)

// --- Generic

type Node interface {
	fs.Node
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
	New(client vfs_api.FileSystemServiceClient) (DirectoryNodeService, error)
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

	GetIdentifier() uint64
}

// --- File

type FileNodeServiceFactory interface {
	New(client vfs_api.FileSystemServiceClient) (FileNodeService, error)
}

type FileNodeService interface {
	New(identifier uint64, size uint64) (FileNode, error)
	Close() error
}

type FileNode interface {
	Node

	GetIdentifier() uint64
	GetSize() uint64
}

