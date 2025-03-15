package factory

import (
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/directory/node/service"
	streamable_node_service_factory "fuse_video_steamer/filesystem/server/provider/fuse/filesystem/streamable/node/service/factory"
	file_node_service_factory "fuse_video_steamer/filesystem/server/provider/fuse/filesystem/file/node/service/factory"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {}

var _ interfaces.DirectoryNodeServiceFactory = &Factory{}
func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(apiClient api.FileSystemServiceClient) (interfaces.DirectoryNodeService, error) {
	directoryNodeServiceFactory := New()
	streamableNodeServiceFactory := streamable_node_service_factory.New()
	fileNodeServiceFactory := file_node_service_factory.New()

	return service.New(apiClient, directoryNodeServiceFactory, streamableNodeServiceFactory, fileNodeServiceFactory)
}
