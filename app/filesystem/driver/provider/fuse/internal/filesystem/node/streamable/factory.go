package streamable

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	"fuse_video_streamer/config"

	handle_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/streamable"
)

type Factory struct {
	config *config.Config
	loggerFactory interfaces_logger.LoggerFactory
}

var _ node.StreamableNodeServiceFactory = &Factory{}

func NewFactory(config *config.Config, loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		config: config,
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(client interfaces_filesystem_client.Client, tree node.Tree) (node.StreamableNodeService, error) {
	streamableNodeService, err := factory.loggerFactory.NewLogger("Streamable Node Service")
	if err != nil {
		return nil, err
	}

	streamableHandleServiceFactory := handle_streamable.NewFactory(factory.config, factory.loggerFactory)

	return NewService(client, streamableHandleServiceFactory, factory.loggerFactory, streamableNodeService, tree)
}
