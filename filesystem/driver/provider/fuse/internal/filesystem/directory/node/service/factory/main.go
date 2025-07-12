package factory

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	file_node_service_factory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/node/service/factory"
	streamable_node_service_factory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node/service/factory"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
)

type Factory struct {}

var _ interfaces.DirectoryNodeServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(client filesystem_client_interfaces.Client) (interfaces.DirectoryNodeService, error) {
	directoryNodeServiceFactory := New()
	streamableNodeServiceFactory := streamable_node_service_factory.New()
	fileNodeServiceFactory := file_node_service_factory.New()

	return service.New(client, directoryNodeServiceFactory, streamableNodeServiceFactory, fileNodeServiceFactory)
}
