package filesystem

import (
	"fuse_video_steamer/fuse/node"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"

	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

type FileSystem struct {
	fileSystem *vfs.FileSystem
	logger     *zap.SugaredLogger
}

func New(fileSystem *vfs.FileSystem) *FileSystem {
	sugaredLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &FileSystem{
		fileSystem: fileSystem,
		logger:     sugaredLogger,
	}

}

var _ fs.FS = &FileSystem{}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	vfsRoot := fileSystem.fileSystem.GetRoot()

	root := node.NewDirectory(fileSystem.fileSystem, vfsRoot.GetNode().GetIdentifier())

	return root, nil
}
