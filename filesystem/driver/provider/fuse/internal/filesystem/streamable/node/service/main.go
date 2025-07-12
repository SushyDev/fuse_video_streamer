package service

import (
	"fmt"
	"sync"
	"sync/atomic"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/logger"
)

type Service struct {
	client   filesystem_client_interfaces.Client
	logger   *logger.Logger
	registry *registry.Registry

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces.StreamableNodeService = &Service{}

func New(client filesystem_client_interfaces.Client, logger *logger.Logger) (interfaces.StreamableNodeService, error) {
	registry := registry.GetInstance(client)

	return &Service{
		client:   client,
		logger:   logger,
		registry: registry,
	}, nil
}

func (service *Service) New(identifier uint64) (interfaces.StreamableNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.IsClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		message := fmt.Sprintf("Failed to create logger for streamable node with identifier %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	fileSystem := service.client.GetFileSystem()

	size, err := fileSystem.GetFileInfo(identifier)

	if err != nil {
		message := fmt.Sprintf("Failed to get video size for %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	newNode, err := node.New(service.client, logger, identifier, size)
	if err != nil {
		service.logger.Error("Failed to create new streamable node", err)
		return nil, err
	}

	service.registry.Add(newNode)

	return newNode, nil
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
