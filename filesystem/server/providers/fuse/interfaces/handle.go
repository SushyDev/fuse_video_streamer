package interfaces

import (
	"fuse_video_steamer/vfs_api"
	"io"

	"github.com/anacrolix/fuse/fs"
)

// --- Root

type RootHandleServiceFactory interface {
	New() (RootHandleService, error)
}

type RootHandleService interface {
	New() (RootHandle, error)
}

type RootHandle interface {
	fs.Handle
	fs.HandleReadDirAller
	io.Closer
}

// --- Directory

type DirectoryHandleServiceFactory interface {
	New(DirectoryNode, vfs_api.FileSystemServiceClient) (DirectoryHandleService, error)
}

type DirectoryHandleService interface {
	New() (DirectoryHandle, error)
}

type DirectoryHandle interface {
	fs.Handle
	fs.HandleReadDirAller
	io.Closer
}

// --- File

type FileHandleServiceFactory interface {
	New(FileNode, vfs_api.FileSystemServiceClient) (FileHandleService, error)
}

type FileHandleService interface {
	New() (FileHandle, error)
}

type FileHandle interface {
	fs.Handle
	fs.HandleReader
	fs.HandleReleaser
	io.Closer

	GetIdentifier() uint64
}
