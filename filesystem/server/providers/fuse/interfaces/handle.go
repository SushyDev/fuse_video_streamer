package interfaces

import (
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse/fs"
)

// --- Root

type RootHandleServiceFactory interface {
	New() (RootHandleService, error)
}

type RootHandleService interface {
	New() (RootHandle, error)
	Close() error
}

type RootHandle interface {
	fs.Handle
	fs.HandleReadDirAller

	Close() error
}

// --- Directory

type DirectoryHandleServiceFactory interface {
	New(DirectoryNode, vfs_api.FileSystemServiceClient) (DirectoryHandleService, error)
}

type DirectoryHandleService interface {
	New() (DirectoryHandle, error)
	Close() error
}

type DirectoryHandle interface {
	fs.Handle
	fs.HandleReadDirAller

	GetIdentifier() uint64
	Close() error
}

// --- File

type FileHandleServiceFactory interface {
	New(FileNode, vfs_api.FileSystemServiceClient) (FileHandleService, error)
}

type FileHandleService interface {
	New() (FileHandle, error)
	Close() error
}

type FileHandle interface {
	fs.Handle
	fs.HandleReader
	fs.HandleReleaser

	GetIdentifier() uint64
	Close() error
}
