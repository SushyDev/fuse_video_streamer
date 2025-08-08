package file

import (
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	handle_file "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/file"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_node.FileNodeServiceFactory = &Factory{}

func NewFactory(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(client factory_filesystem_client.Client, tree interfaces_node.Tree) (interfaces_node.FileNodeService, error) {
	fileNodeServiceLogger, err := factory.loggerFactory.NewLogger("File Node Service")
	if err != nil {
		return nil, err
	}

	fileHandleServiceFactory := handle_file.NewFactory(factory.loggerFactory)

	return NewService(client, fileHandleServiceFactory, factory.loggerFactory, fileNodeServiceLogger, tree)
}
