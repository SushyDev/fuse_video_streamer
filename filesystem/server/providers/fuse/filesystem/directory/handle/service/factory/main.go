package factory

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/handle/service"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/vfs_api"
)

type Factory struct {}

var _ interfaces.DirectoryHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.DirectoryNode, apiClient vfs_api.FileSystemServiceClient) (interfaces.DirectoryHandleService, error) {
	return service.New(node, apiClient), nil
}

