package directory

import (
	"fuse_video_streamer/config"
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	node_file "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/file"
	node_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/streamable"
)

type Factory struct {
	config        *config.Config
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_node.DirectoryNodeServiceFactory = &Factory{}

func NewFactory(config *config.Config, loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		config:        config,
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(client interfaces_filesystem_client.Client, tree interfaces_node.Tree) (interfaces_node.DirectoryNodeService, error) {
	directoryNodeServiceFactory := NewFactory(factory.config, factory.loggerFactory)
	streamableNodeServiceFactory := node_streamable.NewFactory(factory.config, factory.loggerFactory)
	fileNodeServiceFactory := node_file.NewFactory(factory.loggerFactory)

	logger, err := factory.loggerFactory.NewLogger("Directory Node Service")
	if err != nil {
		return nil, err
	}

	return NewService(
		client,
		directoryNodeServiceFactory,
		streamableNodeServiceFactory,
		fileNodeServiceFactory,
		factory.loggerFactory,
		logger,
		tree,
	)
}
