package service

import (
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_stream "fuse_video_streamer/stream/factory"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/handle"
)

type Service struct {
	node          interfaces_fuse.StreamableNode
	client        interfaces_filesystem_client.Client
	loggerFactory interfaces_logger.LoggerFactory
	streamFactory *factory_stream.Factory

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_fuse.StreamableHandleService = &Service{}

func New(
	node interfaces_fuse.StreamableNode,
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

func (service *Service) New() (interfaces_fuse.StreamableHandle, error) {
	if service.IsClosed() {
		service.logger.Warn("Attempted to create a new Streamable Handle after service was closed")
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("File Handle")
	if err != nil {
		service.logger.Error("Failed to create logger for Streamable Handle", err)
		return nil, err
	}

	stream, err := service.streamFactory.NewStream(service.node.GetIdentifier(), service.node.GetSize())
	if err != nil {
		service.logger.Error("Failed to create stream for Streamable Handle", err)
		return nil, err
	}

	return handle.New(service.node, stream, logger), nil
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
