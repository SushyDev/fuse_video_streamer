package interfaces

import (
	"github.com/anacrolix/fuse/fs"
)

type FuseFileSystemService interface {
	New() FuseFileSystem
}

type FuseFileSystem interface {
	fs.FS
	// fs.FSDestroyer

	Close() error
}
