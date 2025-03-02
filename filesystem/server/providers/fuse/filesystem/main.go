package filesystem

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/providers/fuse/registry"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	rootNodeService interfaces.RootNodeService

	registry *registry.Registry

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

		registry: registry.GetInstance(),

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
	fileSystem.logger.Info("Closing")

	fileSystem.rootNodeService.Close()
	fileSystem.rootNodeService = nil

	fileSystem.registry.CloseNodes()
	fileSystem.registry = nil

	fileSystem.logger.Info("Closed")

	return nil
}
