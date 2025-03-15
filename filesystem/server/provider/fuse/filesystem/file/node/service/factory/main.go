package factory

import (
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/file/node/service"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"

	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {}

var _ interfaces.FileNodeServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(client api.FileSystemServiceClient) (interfaces.FileNodeService, error) {
	logger, err := logger.NewLogger("File Node Service")
	if err != nil {
		return nil, err
	}

	return service.New(client, logger)
}
