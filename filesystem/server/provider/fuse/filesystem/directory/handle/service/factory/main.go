package factory

import (
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/directory/handle/service"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {}

var _ interfaces.DirectoryHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.DirectoryNode, apiClient api.FileSystemServiceClient) (interfaces.DirectoryHandleService, error) {
	return service.New(node, apiClient), nil
}

