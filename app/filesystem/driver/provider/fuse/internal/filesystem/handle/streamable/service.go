package streamable

import (
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	factory_stream "fuse_video_streamer/stream/drivers/http_ring_buffer/factory"
)

type Service struct {
	node          interfaces_node.StreamableNode
	client        interfaces_filesystem_client.Client
	loggerFactory interfaces_logger.LoggerFactory
	streamFactory *factory_stream.Factory

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_handle.StreamableHandleService = &Service{}

func NewService(
	node interfaces_node.StreamableNode,
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
	streamFactory *factory_stream.Factory,
	logger interfaces_logger.Logger,
) *Service {
	return &Service{
		node:          node,
		client:        client,
		loggerFactory: loggerFactory,
		streamFactory: streamFactory,
		logger:        logger,
	}
}

func (service *Service) NewHandle() (interfaces_handle.StreamableHandle, error) {
	if service.IsClosed() {
		service.logger.Warn("Attempted to create a new Streamable Handle after service was closed")
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("File Handle")
	if err != nil {
		service.logger.Error("failed to create logger for Streamable Handle", err)
		return nil, err
	}

	stream, err := service.streamFactory.NewStream(service.node.GetRemoteIdentifier(), service.node.GetSize())
	if err != nil {
		service.logger.Error("failed to create stream for Streamable Handle", err)
		return nil, err
	}

	return NewHandle(service.node, stream, logger), nil
}

func (service *Service) Close() error {
	if !service.closed.CompareAndSwap(false, true) {
		return nil
	}

	service.streamFactory.Close()

	return nil
}

func (service *Service) IsClosed() bool {
	return service.closed.Load()
}
