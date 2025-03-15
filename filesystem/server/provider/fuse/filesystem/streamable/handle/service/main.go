package service

import (
	"context"

	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/streamable/handle"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/stream/factory"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"
)

type Service struct {
	node      interfaces.FileNode
	apiClient api.FileSystemServiceClient

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.StreamableHandleService = &Service{}

func New(node interfaces.StreamableNode, apiClient api.FileSystemServiceClient) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		node: node,
		apiClient: apiClient,

		ctx: ctx,
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


	streamFactory := factory.NewFactory(service.apiClient, service.node.GetIdentifier(), service.node.GetSize())

	stream, err := streamFactory.NewStream()
	if err != nil {
		return nil, err
	}

	return handle.New(service.node, stream, logger), nil
}

func (service *Service) Close() error {
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
