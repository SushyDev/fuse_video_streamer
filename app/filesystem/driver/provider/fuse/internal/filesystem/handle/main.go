package handle

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse_filesystem_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"github.com/anacrolix/fuse/fs"
)

// Handle

type handle interface {
	interfaces_fuse.UseClosable

	fs.Handle
}

type RootHandle interface {
	handle

	fs.HandleReadDirAller
}

type DirectoryHandle interface {
	handle

	interfaces_fuse.UseIdentifier

	fs.HandleReadDirAller
}

type FileHandle interface {
	handle

	interfaces_fuse.UseIdentifier

	fs.Handle
	fs.HandleReadAller
	fs.HandleReader
	fs.HandleWriter
	fs.HandleReleaser
	fs.HandleFlusher

	fs.NodeFsyncer
}

type StreamableHandle interface {
	handle

	interfaces_fuse.UseIdentifier

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser
}

// Service

type RootHandleService interface {
	interfaces_fuse.UseClosable

	NewHandle() (RootHandle, error)
}

type DirectoryHandleService interface {
	interfaces_fuse.UseClosable

	NewHandle(interfaces_fuse_filesystem_node.DirectoryNode) (DirectoryHandle, error)
}

type FileHandleService interface {
	interfaces_fuse.UseClosable

	NewHandle(interfaces_fuse_filesystem_node.FileNode) (FileHandle, error)
}

type StreamableHandleService interface {
	interfaces_fuse.UseClosable

	NewHandle() (StreamableHandle, error)
}

// Factory

type RootHandleServiceFactory interface {
	NewService() (RootHandleService, error)
}

type DirectoryHandleServiceFactory interface {
	NewService() DirectoryHandleService
}

type FileHandleServiceFactory interface {
	NewService() FileHandleService
}

type StreamableHandleServiceFactory interface {
	NewService(interfaces_fuse_filesystem_node.StreamableNode, interfaces_filesystem_client.Client) (StreamableHandleService, error)
}
