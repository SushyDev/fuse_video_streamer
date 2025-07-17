package interfaces

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"

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
	New() DirectoryHandleService
}

type DirectoryHandleService interface {
	useClosable

	New(DirectoryNode) (DirectoryHandle, error)
}

type DirectoryHandle interface {
	useIntentifier

	Handle

	fs.Handle
	fs.HandleReadDirAller
}

// --- Streamable

type StreamableHandleServiceFactory interface {
	New(StreamableNode, interfaces_filesystem_client.Client) (StreamableHandleService, error)
}

type StreamableHandleService interface {
	useClosable

	New() (StreamableHandle, error)
}

type StreamableHandle interface {
	useIntentifier

	Handle

	fs.Handle
	fs.HandleReader
	fs.HandleReleaser
}

// --- File

type FileHandleServiceFactory interface {
	New() FileHandleService
}

type FileHandleService interface {
	useClosable

	New(FileNode) (FileHandle, error)
}

type FileHandle interface {
	useIntentifier

	Handle

	fs.Handle
	fs.HandleReadAller
	fs.HandleReader
	fs.HandleWriter
	fs.HandleReleaser
	fs.HandleFlusher

	fs.NodeFsyncer
}
