package service

import (
	"context"
	"fmt"
	"sync"

	filesystem_client_interfaces "fuse_video_steamer/filesystem/client/interfaces"
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/file/node"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/provider/fuse/registry"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"
)

type Service struct {
	client   filesystem_client_interfaces.Client
	logger   *logger.Logger
	registry *registry.Registry

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileNodeService = &Service{}

var clients = []api.FileSystemServiceClient{}

func New(client filesystem_client_interfaces.Client, logger *logger.Logger) (interfaces.FileNodeService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := registry.GetInstance(client)

	return &Service{
		client:   client,
		logger:   logger,
		registry: registry,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (service *Service) New(identifier uint64) (interfaces.FileNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.isClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		panic(err)
	}

	fileSystem := service.client.GetFileSystem()

	size, err := fileSystem.GetFileInfo(identifier)
	if err != nil {
		message := fmt.Sprintf("Failed to get video size for %d", identifier)
		service.logger.Error(message, err)
		return nil, err
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
