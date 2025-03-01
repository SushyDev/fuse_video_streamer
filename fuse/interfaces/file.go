package interfaces

import (
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse/fs"
)

type NodeService interface {
	NewRoot() (Root, error)
	NewDirectory(client vfs_api.FileSystemServiceClient, identifier uint64) (Directory, error)
	NewFile(client vfs_api.FileSystemServiceClient, identifier uint64, size uint64) (File, error)
}

type Root interface {
	fs.Handle
	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
	fs.HandleReadDirAller
}

type File interface {
	fs.Handle
	fs.Node
	fs.HandleReader
	fs.HandleFlusher
}

type Directory interface {
	fs.Handle
	fs.Node
	fs.NodeOpener
	fs.NodeRequestLookuper
	fs.HandleReadDirAller
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeLinker
}
