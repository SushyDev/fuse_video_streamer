package service

import (
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node"
)

type Service struct {
	fileSystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_fuse.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory

	loggerFactory interfaces_logger.LoggerFactory

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_fuse.RootNodeService = &Service{}

func New(
	filesystemClientRepository interfaces_filesystem_client.ClientRepository,

	directoryNodeServiceFactory interfaces_fuse.DirectoryNodeServiceFactory,
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory,

	loggerFactory interfaces_logger.LoggerFactory,

	logger interfaces_logger.Logger,
) *Service {
	return &Service{
		fileSystemClientRepository: filesystemClientRepository,

		directoryNodeServiceFactory:   directoryNodeServiceFactory,
		directoryHandleServiceFactory: directoryHandleServiceFactory,

		loggerFactory: loggerFactory,

		logger: logger,
	}
}

func (service *Service) New() (interfaces_fuse.RootNode, error) {
	if service.IsClosed() {
		service.logger.Warn("Root Node Service is closed, cannot create new root node")
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("Root Node")
	if err != nil {
		service.logger.Error("Failed to create logger for Root Node", err)
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(nil)
	if err != nil {
		service.logger.Error("Failed to create Directory Node Service for Root Node", err)
		return nil, err
	}

	return node.New(
		service.fileSystemClientRepository,
		service.directoryNodeServiceFactory,
		service.directoryHandleServiceFactory,
		service.loggerFactory,
		directoryNodeService,
		logger,
	)
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
