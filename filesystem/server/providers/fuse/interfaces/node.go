package interfaces

import (
	"io"

	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse/fs"
)

// --- Root

type RootNodeServiceFactory interface {
	New() (RootNodeService, error)
}

type RootNodeService interface {
	New() (RootNode, error)
	io.Closer
}

type RootNode interface {
	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
}

// --- Directory

type DirectoryNodeServiceFactory interface {
	New(client vfs_api.FileSystemServiceClient) (DirectoryNodeService, error)
}

type DirectoryNodeService interface {
	New(identifier uint64) (DirectoryNode, error)
	io.Closer
}

type DirectoryNode interface {
	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeLinker
	io.Closer

	GetIdentifier() uint64
}

// --- File

type FileNodeServiceFactory interface {
	New(client vfs_api.FileSystemServiceClient) (FileNodeService, error)
}

type FileNodeService interface {
	New(identifier uint64, size uint64) (FileNode, error)
	io.Closer
}

type FileNode interface {
	fs.Node
	io.Closer

	GetIdentifier() uint64
	GetSize() uint64
}

