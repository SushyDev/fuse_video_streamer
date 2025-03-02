package filesystem

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	rootNodeService interfaces.RootNodeService

	logger  *logger.Logger
}

var _ interfaces.FuseFileSystem = &FileSystem{}

func New(rootNodeService interfaces.RootNodeService) interfaces.FuseFileSystem {
	logger, err := logger.NewLogger("Filesystem")
	if err != nil {
		panic(err)
	}

	return &FileSystem{
		rootNodeService: rootNodeService,

		logger: logger,
	}
}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	return fileSystem.rootNodeService.New()
}

// func (fileSystem *FileSystem) Destroy() {
// 	fmt.Println("\nDestroying filesystem\n")
// }

func (fileSystem *FileSystem) Close() error {
	fileSystem.logger.Info("Closing filesystem")

	fileSystem.rootNodeService.Close()

	fileSystem.logger.Info("Closed filesystem")

	return nil
}
