package service

import (
	"context"
	"fmt"
	"sync"

	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/file/node"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/providers/fuse/registry"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	client vfs_api.FileSystemServiceClient

	registry *registry.Registry

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileNodeService = &Service{}

var clients = []vfs_api.FileSystemServiceClient{}

func New(client vfs_api.FileSystemServiceClient) (interfaces.FileNodeService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := registry.GetInstance()

	return &Service{
		client: client,
		registry: registry,

		ctx: ctx,
		cancel: cancel,
	}, nil
}

func (service *Service) New(identifier uint64, size uint64) (interfaces.FileNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.isClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		panic(err)
	}

	newNode := node.New(service.client, logger, identifier, size)

	service.registry.Add(newNode)

	return newNode, nil
}

func (service *Service) Close() error {
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
