package service

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/logger"
	"sync/atomic"
)

type Service struct {
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory

	closed atomic.Bool
}

var _ interfaces.RootNodeService = &Service{}

func New(directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory) *Service {
	return &Service{
		directoryNodeServiceFactory: directoryNodeServiceFactory,
	}
}

func (service *Service) New() (interfaces.RootNode, error) {
	if service.IsClosed() {
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
	if !service.closed.CompareAndSwap(false, true) {
		return nil
	}

	return nil
}

func (service *Service) IsClosed() bool {
	return service.closed.Load()
}
