package factory

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/handle/service"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/vfs_api"
)

type Factory struct {}

var _ interfaces.FileHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.FileNode, apiClient vfs_api.FileSystemServiceClient) (interfaces.FileHandleService, error) {
	service := service.New(node, apiClient)

	return service, nil
}

