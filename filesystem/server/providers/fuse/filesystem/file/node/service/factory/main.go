package factory

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/node/service"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/vfs_api"
)

type Factory struct {}

var _ interfaces.FileNodeServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(client vfs_api.FileSystemServiceClient) (interfaces.FileNodeService, error) {
	return service.New(client)
}
