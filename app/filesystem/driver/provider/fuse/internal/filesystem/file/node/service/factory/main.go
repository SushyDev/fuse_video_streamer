package factory

import (
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	factory_file_handle_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/handle/service/factory"

	service_file_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/node/service"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.FileNodeServiceFactory = &Factory{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) New(client factory_filesystem_client.Client, tree interfaces_fuse.Tree) (interfaces_fuse.FileNodeService, error) {
	fileNodeServiceLogger, err := factory.loggerFactory.NewLogger("File Node Service")
	if err != nil {
		return nil, err
	}

	fileHandleServiceFactory := factory_file_handle_service.New(factory.loggerFactory)

	return service_file_node.New(client, fileHandleServiceFactory, factory.loggerFactory, fileNodeServiceLogger, tree)
}
