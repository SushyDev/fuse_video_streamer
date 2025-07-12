package interfaces

import (
	"github.com/anacrolix/fuse/fs"
)

type FuseFileSystemService interface {
	New() FuseFileSystem
}

type FuseFileSystem interface {
	useClosable

	fs.FS
	// fs.FSDestroyer
}
