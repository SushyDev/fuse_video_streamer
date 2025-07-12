package service

import (
	"fmt"
	"sync"
	"sync/atomic"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/logger"
)

type Service struct {
	client filesystem_client_interfaces.Client

	directoryNodeServiceFactory  interfaces.DirectoryNodeServiceFactory
	streamableNodeServiceFactory interfaces.StreamableNodeServiceFactory
	fileNodeServiceFactory       interfaces.FileNodeServiceFactory

	registry *registry.Registry

	logger *logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces.DirectoryNodeService = &Service{}

func New(
	client filesystem_client_interfaces.Client,
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory,
	streamableNodeServiceFactory interfaces.StreamableNodeServiceFactory,
	fileNodeServiceFactory interfaces.FileNodeServiceFactory,
) (interfaces.DirectoryNodeService, error) {
	registry := registry.GetInstance(client)

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		return nil, err
	}

	service := &Service{
		client: client,

		directoryNodeServiceFactory:  directoryNodeServiceFactory,
		streamableNodeServiceFactory: streamableNodeServiceFactory,
		fileNodeServiceFactory:       fileNodeServiceFactory,

		registry: registry,

		logger: logger,
	}

	return service, nil
}

func (service *Service) New(identifier uint64) (interfaces.DirectoryNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.IsClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		service.logger.Error("Failed to create logger for new directory node", err)
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("Failed to create directory node service", err)
		return nil, err
	}

	streamableNodeService, err := service.streamableNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("Failed to create streamable node service", err)
		return nil, err
	}

	fileNodeService, err := service.fileNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("Failed to create file node service", err)
		return nil, err
	}

	newNode, err := node.New(directoryNodeService, streamableNodeService, fileNodeService, service.client, logger, identifier)
	if err != nil {
		service.logger.Error("Failed to create new directory node", err)
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
