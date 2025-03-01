package filesystem

import (
	"fuse_video_steamer/fuse/registry"
	"fuse_video_steamer/fuse/service"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	service *service.Service
	logger  *logger.Logger
}

func New() *FileSystem {
	logger, err := logger.NewLogger("Filesystem")
	if err != nil {
		panic(err)
	}

	service := service.NewService()

	return &FileSystem{
		service: service,
		logger: logger,
	}

}

var _ fs.FS = &FileSystem{}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	root, err := fileSystem.service.NewRoot()
	if err != nil {
		return nil, err
	}

	return root, nil
}

// var _ fs.FSDestroyer = &FileSystem{}
//
// func (fileSystem *FileSystem) Destroy() {
// 	fmt.Println("\nDestroying filesystem\n")
// }

func (fileSystem *FileSystem) Close() {
	fileSystem.logger.Info("Closing filesystem")

	fileSystem.service.Close()

	registry := registry.GetInstance()
	registry.CloseNodes()

	fileSystem.logger.Info("Closed filesystem")
}
