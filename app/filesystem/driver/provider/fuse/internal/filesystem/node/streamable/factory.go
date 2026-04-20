package streamable

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	handle_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/streamable"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	stream_factory "fuse_video_streamer/stream/drivers/http_ring_buffer/factory"
)

type Factory struct {
	config        *config.Config
	loggerFactory interfaces_logger.LoggerFactory
	registry      registry.Registry
}

var _ node.StreamableNodeServiceFactory = &Factory{}

func NewFactory(config *config.Config, loggerFactory interfaces_logger.LoggerFactory, registry registry.Registry) *Factory {
	return &Factory{
		config:        config,
		loggerFactory: loggerFactory,
		registry:      registry,
	}
}

func (factory *Factory) NewService(client interfaces_filesystem_client.Client, tree node.Tree) (node.StreamableNodeService, error) {
	streamableNodeServiceLogger, err := factory.loggerFactory.NewLogger("Streamable Node Service")
	if err != nil {
		return nil, err
	}

	sf := stream_factory.New(factory.config, client, factory.loggerFactory)
	bp := pool.NewBufferPool()
	streamableHandleServiceFactory := handle_streamable.NewFactory(sf, bp, factory.loggerFactory)

	return NewService(client, streamableHandleServiceFactory, factory.loggerFactory, streamableNodeServiceLogger, tree, factory.registry)
}
