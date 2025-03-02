package service

import (
	"context"

	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/handle"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	node interfaces.DirectoryNode
	apiClient     vfs_api.FileSystemServiceClient

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryHandleService = &Service{}

func New(node interfaces.DirectoryNode, apiClient vfs_api.FileSystemServiceClient) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		node: node,
		apiClient: apiClient,

		ctx: ctx,
		cancel: cancel,
	}
}

func (service *Service) New() (interfaces.DirectoryHandle, error) {
	if service.isClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("Directory Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(service.apiClient, service.node, logger), nil
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
