package filesystem

import (
	"fuse_video_steamer/fuse/node"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	logger *logger.Logger
}

func New() *FileSystem {
	logger, err := logger.NewLogger("File System")
	if err != nil {
		panic(err)
	}

	return &FileSystem{
		logger: logger,
	}

}

var _ fs.FS = &FileSystem{}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	root := node.NewRoot()

	return root, nil
}
