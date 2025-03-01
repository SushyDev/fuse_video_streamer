package service

import (
	"context"
	"fmt"
	"sync"

	"fuse_video_steamer/api_clients"
	"fuse_video_steamer/fuse/interfaces"
	"fuse_video_steamer/fuse/node"
	"fuse_video_steamer/fuse/registry"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/stream/factory"
	"fuse_video_steamer/stream/manager"
	"fuse_video_steamer/vfs_api"
)

type Service struct {
	stream_manager *manager.Manager
	registry *registry.Registry

	mu sync.RWMutex

	ctx context.Context
	cancel context.CancelFunc
}

var _ interfaces.NodeService = &Service{}

var clients = []vfs_api.FileSystemServiceClient{}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		stream_manager: manager.GetInstance(),
		registry: registry.GetInstance(),

		ctx: ctx,
		cancel: cancel,
	}
}

func (service *Service) NewRoot() (interfaces.Root, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.IsClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		panic(err)
	}

	clients := api_clients.GetClients()

	root := node.NewRoot(service, logger, clients)

	return root, nil
}

func (service *Service) NewDirectory(client vfs_api.FileSystemServiceClient, identifier uint64) (interfaces.Directory, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.IsClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Directory Node")
	if err != nil {
		panic(err)
	}

	directory := node.NewDirectory(service, client, logger, identifier)

	service.registry.AddDirectory(identifier, directory)

	return directory, nil
}

func (service *Service) NewFile(client vfs_api.FileSystemServiceClient, identifier uint64, size uint64) (interfaces.File, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.IsClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("File Node")
	if err != nil {
		panic(err)
	}

	streamFactory := factory.NewFactory(client, identifier, size)

	service.stream_manager.AddFactory(identifier, streamFactory)

	file := node.NewFile(client, logger, streamFactory, identifier, size)

	service.registry.AddFile(identifier, file)

	return file, nil
}


func (service *Service) Close() {
	service.mu.Lock()
	defer service.mu.Unlock()

	service.cancel()
}

func (service *Service) IsClosed() bool {
	select {
	case <-service.ctx.Done():
		return true
	default:
		return false
	}
}

