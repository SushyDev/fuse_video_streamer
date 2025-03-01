package filesystem

import (
	fuse_interfaces "fuse_video_steamer/fuse/interfaces"
	fvs_fuse_node_service "fuse_video_steamer/fuse/node/service"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FuseFileSystem struct {
	nodeService fuse_interfaces.NodeService
	logger  *logger.Logger
}

var _ fuse_interfaces.FuseFileSystem = &FuseFileSystem{}

func New() fuse_interfaces.FuseFileSystem {
	logger, err := logger.NewLogger("Filesystem")
	if err != nil {
		panic(err)
	}

	nodeService := fvs_fuse_node_service.New()

	return &FuseFileSystem{
		nodeService: nodeService,
		logger: logger,
	}
}

func (fileSystem *FuseFileSystem) Root() (fs.Node, error) {
	root, err := fileSystem.nodeService.NewRoot()
	if err != nil {
		return nil, err
	}

	return root, nil
}

// func (fileSystem *FileSystem) Destroy() {
// 	fmt.Println("\nDestroying filesystem\n")
// }

func (fileSystem *FuseFileSystem) Close() error {
	fileSystem.logger.Info("Closing filesystem")

	fileSystem.nodeService.Close()

	fileSystem.logger.Info("Closed filesystem")

	return nil
}
