package service

import (
	"context"

	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/handle"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/stream/factory"
	"fuse_video_steamer/cache"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	node      interfaces.FileNode
	apiClient vfs_api.FileSystemServiceClient

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileHandleService = &Service{}

func New(node interfaces.FileNode, apiClient vfs_api.FileSystemServiceClient) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		node: node,
		apiClient: apiClient,

		ctx: ctx,
		cancel: cancel,
	}
}

func (service *Service) New() (interfaces.FileHandle, error) {
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

	defaultOpts := cache.DefaultCacheOptions()
	cache := cache.NewCache(stream, int64(service.node.GetSize()), defaultOpts)

	return handle.New(cache, logger), nil
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
