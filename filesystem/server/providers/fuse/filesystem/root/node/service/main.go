package service

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/root/node"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory
	apiClient     vfs_api.FileSystemServiceClient
}

var _ interfaces.RootNodeService = &Service{}

func New(directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory) *Service {
	return &Service{
		directoryNodeServiceFactory: directoryNodeServiceFactory,
	}
}

func (service *Service) New() (interfaces.RootNode, error) {
	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(nil)

	return node.New(directoryNodeService, logger)
}

func (service *Service) Close() error {
	return nil
}
