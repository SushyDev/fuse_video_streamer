package service

import (
	"context"
	"fmt"
	"sync"

	"fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/node"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/providers/fuse/registry"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	client vfs_api.FileSystemServiceClient

	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory
	fileNodeServiceFactory interfaces.FileNodeServiceFactory

	registry *registry.Registry

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.DirectoryNodeService = &Service{}

var clients = []vfs_api.FileSystemServiceClient{}

func New(client vfs_api.FileSystemServiceClient, directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory, fileNodeServiceFactory interfaces.FileNodeServiceFactory) (interfaces.DirectoryNodeService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := registry.GetInstance()

	return &Service{
		client: client,

		directoryNodeServiceFactory: directoryNodeServiceFactory,
		fileNodeServiceFactory: fileNodeServiceFactory,

		registry: registry,

		ctx: ctx,
		cancel: cancel,
	}, nil
}

func (service *Service) New(identifier uint64) (interfaces.DirectoryNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.isClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		panic(err)
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(service.client)
	if err != nil {
		return nil, err
	}

	fileNodeService, err := service.fileNodeServiceFactory.New(service.client)
	if err != nil {
		return nil, err
	}

	newNode := node.New(directoryNodeService, fileNodeService, service.client, logger, identifier)

	service.registry.Add(newNode)

	return newNode, nil
}

func (service *Service) Close() error {
	// service.mu.Lock()
	// defer service.mu.Unlock()

	if service.isClosed() {
		return nil
	}

	service.cancel()

	fmt.Println("Directory node service closed")

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
