package service

import (
	"context"

	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/root/node"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/logger"
)

type Service struct {
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.RootNodeService = &Service{}

func New(directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		directoryNodeServiceFactory: directoryNodeServiceFactory,

		ctx: ctx,
		cancel: cancel,
	}
}

func (service *Service) New() (interfaces.RootNode, error) {
	if service.isClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(nil)

	return node.New(directoryNodeService, logger)
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
