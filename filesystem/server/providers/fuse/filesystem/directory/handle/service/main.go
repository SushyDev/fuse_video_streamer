package service

import (
	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/handle"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	node interfaces.DirectoryNode
	apiClient     vfs_api.FileSystemServiceClient
}

var _ interfaces.DirectoryHandleService = &Service{}

func New(node interfaces.DirectoryNode, apiClient vfs_api.FileSystemServiceClient) *Service {
	return &Service{
		node: node,
		apiClient: apiClient,
	}
}

func (service *Service) New() (interfaces.DirectoryHandle, error) {
	logger, err := logger.NewLogger("Directory Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(service.apiClient, service.node, logger), nil
}
