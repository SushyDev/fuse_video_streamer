package streamable

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	stream_factory "fuse_video_streamer/stream/drivers/http_ring_buffer/factory"
)

type Factory struct {
	LoggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.StreamableHandleServiceFactory = &Factory{}

func NewFactory(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		LoggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(node interfaces_node.StreamableNode, client interfaces_filesystem_client.Client) (interfaces_handle.StreamableHandleService, error) {
	streamFactory := stream_factory.New(client, factory.LoggerFactory)

	streamableServiceLogger, err := factory.LoggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}

	service := NewService(node, client, factory.LoggerFactory, streamFactory, streamableServiceLogger)

	return service, nil
}
