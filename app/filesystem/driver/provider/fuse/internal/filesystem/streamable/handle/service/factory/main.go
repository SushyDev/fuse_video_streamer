package factory

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	stream_factory "fuse_video_streamer/stream/drivers/http_ring_buffer/factory"

	service_streamable_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/handle/service"
)

type Factory struct {
	LoggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.StreamableHandleServiceFactory = &Factory{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		LoggerFactory: loggerFactory,
	}
}

func (factory *Factory) New(node interfaces_fuse.StreamableNode, client interfaces_filesystem_client.Client) (interfaces_fuse.StreamableHandleService, error) {
	streamFactory := stream_factory.New(client, factory.LoggerFactory)

	streamableServiceLogger, err := factory.LoggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}

	service := service_streamable_service.New(node, client, factory.LoggerFactory, streamFactory, streamableServiceLogger)

	return service, nil
}
