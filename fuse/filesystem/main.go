package filesystem

import (
	"fuse_video_steamer/fuse/node"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

type FileSystem struct {
	logger *zap.SugaredLogger
}

func New() *FileSystem {
	sugaredLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &FileSystem{
		logger: sugaredLogger,
	}

}

var _ fs.FS = &FileSystem{}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	root := node.NewRoot()

	return root, nil
}
