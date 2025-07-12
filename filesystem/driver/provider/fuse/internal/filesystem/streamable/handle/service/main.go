package service

import (
	"sync/atomic"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/handle"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/stream/factory"
)

type Service struct {
	node   interfaces.StreamableNode
	client filesystem_client_interfaces.Client
	streamFactory *factory.Factory

	closed atomic.Bool
}

var _ interfaces.StreamableHandleService = &Service{}

func New(
	node interfaces.StreamableNode,
	client filesystem_client_interfaces.Client,
	streamFactory *factory.Factory,
) *Service {
	return &Service{
		node:   node,
		client: client,
		streamFactory: streamFactory,
	}
}

func (service *Service) New() (interfaces.StreamableHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("File Handle")
	if err != nil {
		return nil, err
	}

	stream, err := service.streamFactory.NewStream(service.node.GetIdentifier(), service.node.GetSize())
	if err != nil {
		return nil, err
	}

	return handle.New(service.node, stream, logger), nil
}

func (service *Service) Close() error {
	if !service.closed.CompareAndSwap(false, true) {
		return nil
	}

	service.streamFactory.Close()
	service.streamFactory = nil

	return nil
}

func (service *Service) IsClosed() bool {
	return service.closed.Load()
}
