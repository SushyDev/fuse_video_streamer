package service

import (
	"context"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/streamable/handle"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/stream/factory"
)

type Service struct {
	node   interfaces.StreamableNode
	client filesystem_client_interfaces.Client
	streamFactory *factory.Factory

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.StreamableHandleService = &Service{}

func New(
	node interfaces.StreamableNode,
	client filesystem_client_interfaces.Client,
	streamFactory *factory.Factory,
) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		node:   node,
		client: client,
		streamFactory: streamFactory,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (service *Service) New() (interfaces.StreamableHandle, error) {
	if service.isClosed() {
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
	service.streamFactory.Close()
	service.streamFactory = nil

	service.cancel()

	return nil
}

func (service *Service) isClosed() bool {
	select {
	case <-service.ctx.Done():
		return true
	default:
		return false
	}
}
