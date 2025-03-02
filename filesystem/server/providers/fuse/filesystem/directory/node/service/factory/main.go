package factory

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/node/service"
	file_node_service_factory "fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/node/service/factory"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/vfs_api"
)

type Factory struct {}

var _ interfaces.DirectoryNodeServiceFactory = &Factory{}
func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(apiClient vfs_api.FileSystemServiceClient) (interfaces.DirectoryNodeService, error) {
	directoryNodeServiceFactory := New()
	fileNodeServiceFactory := file_node_service_factory.New()

	return service.New(apiClient, directoryNodeServiceFactory, fileNodeServiceFactory)
}
