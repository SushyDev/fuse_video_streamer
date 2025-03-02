package interfaces

import (
	"io"

	"github.com/anacrolix/fuse/fs"
)

type FuseFileSystemService interface {
	New() FuseFileSystem
}

type FuseFileSystem interface {
	fs.FS
	// fs.FSDestroyer
	io.Closer
}
