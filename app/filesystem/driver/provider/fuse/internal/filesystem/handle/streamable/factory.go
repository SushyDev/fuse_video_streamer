package streamable

import (
	"fuse_video_streamer/config"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	stream_factory "fuse_video_streamer/stream/drivers/http_ring_buffer/factory"
)

type Factory struct {
	config        *config.Config
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.StreamableHandleServiceFactory = &Factory{}

func NewFactory(config *config.Config, loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		config:        config,
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(node interfaces_node.StreamableNode, client interfaces_filesystem_client.Client) (interfaces_handle.StreamableHandleService, error) {
	streamFactory := stream_factory.New(factory.config, client, factory.loggerFactory)

	streamableServiceLogger, err := factory.loggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}

	service := NewService(node, client, factory.loggerFactory, streamFactory, streamableServiceLogger)

	return service, nil
}
