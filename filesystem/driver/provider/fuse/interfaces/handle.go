package interfaces

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"

	"github.com/anacrolix/fuse/fs"
)

// --- Generic

type Handle interface {
	useClosable
}

// --- Root

type RootHandleServiceFactory interface {
	New() (RootHandleService, error)
}

type RootHandleService interface {
	useClosable

	New() (RootHandle, error)
}

type RootHandle interface {
	Handle

	fs.Handle
	fs.HandleReadDirAller
}

// --- Directory

type DirectoryHandleServiceFactory interface {
	New(DirectoryNode, filesystem_client_interfaces.Client) (DirectoryHandleService, error)
}

type DirectoryHandleService interface {
	useClosable

	New() (DirectoryHandle, error)
}

type DirectoryHandle interface {
	Handle

	fs.Handle
	fs.HandleReadDirAller

	GetIdentifier() uint64
}

// --- Streamable

type StreamableHandleServiceFactory interface {
	New(StreamableNode, filesystem_client_interfaces.Client) (StreamableHandleService, error)
}

type StreamableHandleService interface {
	useClosable

	New() (StreamableHandle, error)
	Close() error
}

type StreamableHandle interface {
	Handle

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	GetIdentifier() uint64
}

// --- File

type FileHandleServiceFactory interface {
	New(FileNode, filesystem_client_interfaces.Client) (FileHandleService, error)
}

type FileHandleService interface {
	useClosable

	New() (FileHandle, error)
}

type FileHandle interface {
	Handle

	fs.Handle
	fs.HandleReadAller
	fs.HandleReader
	fs.HandleWriter
	fs.HandleReleaser
	fs.HandleFlusher

	fs.NodeFsyncer

	GetIdentifier() uint64
}
