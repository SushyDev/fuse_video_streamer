package factory

import (
	interfacesfilesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_file_node_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/node/service/factory"
	factory_streamable_node_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node/service/factory"

	service_directory_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node/service"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.DirectoryNodeServiceFactory = &Factory{}

func New(loggerFactorr interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactorr,
	}
}

func (factory *Factory) New(client interfacesfilesystem_client.Client) (interfaces_fuse.DirectoryNodeService, error) {
	directoryNodeServiceFactory := New(factory.loggerFactory)
	streamableNodeServiceFactory := factory_streamable_node_service.New(factory.loggerFactory)
	fileNodeServiceFactory := factory_file_node_service.New(factory.loggerFactory)

	logger, err := factory.loggerFactory.NewLogger("Directory Node Service")
	if err != nil {
		return nil, err
	}

	return service_directory_node.New(client, directoryNodeServiceFactory, streamableNodeServiceFactory, fileNodeServiceFactory, factory.loggerFactory, logger)
}
