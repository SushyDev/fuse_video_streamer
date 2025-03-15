package factory

import (
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/file/handle/service"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {}

var _ interfaces.FileHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.FileNode, apiClient api.FileSystemServiceClient) (interfaces.FileHandleService, error) {
	service := service.New(node, apiClient)

	return service, nil
}

