package streamable

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool"
	interfaces_stream "fuse_video_streamer/stream/interfaces"
)

type Factory struct {
	streamFactory interfaces_stream.StreamFactory
	bufferPool    pool.BufferPool
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.StreamableHandleServiceFactory = &Factory{}

func NewFactory(streamFactory interfaces_stream.StreamFactory, bufferPool pool.BufferPool, loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		streamFactory: streamFactory,
		bufferPool:    bufferPool,
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService(node interfaces_node.StreamableNode, client interfaces_filesystem_client.Client) (interfaces_handle.StreamableHandleService, error) {
	streamableServiceLogger, err := factory.loggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}

	service := NewService(node, client, factory.loggerFactory, factory.streamFactory, factory.bufferPool, streamableServiceLogger)

	return service, nil
}
