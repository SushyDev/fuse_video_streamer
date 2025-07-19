package service

import (
	"fmt"
	"sync"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Service struct {
	client interfaces_filesystem_client.Client

	directoryNodeServiceFactory  interfaces_fuse.DirectoryNodeServiceFactory
	streamableNodeServiceFactory interfaces_fuse.StreamableNodeServiceFactory
	fileNodeServiceFactory       interfaces_fuse.FileNodeServiceFactory

	loggerFactory interfaces_logger.LoggerFactory

	registry *registry.Registry

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.DirectoryNodeService = &Service{}

func New(
	client interfaces_filesystem_client.Client,
	directoryNodeServiceFactory interfaces_fuse.DirectoryNodeServiceFactory,
	streamableNodeServiceFactory interfaces_fuse.StreamableNodeServiceFactory,
	fileNodeServiceFactory interfaces_fuse.FileNodeServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	logger interfaces_logger.Logger,
) (interfaces_fuse.DirectoryNodeService, error) {
	registry := registry.GetInstance(client)

	service := &Service{
		client: client,

		directoryNodeServiceFactory:  directoryNodeServiceFactory,
		streamableNodeServiceFactory: streamableNodeServiceFactory,
		fileNodeServiceFactory:       fileNodeServiceFactory,

		loggerFactory: loggerFactory,

		registry: registry,

		logger: logger,
	}

	return service, nil
}

func (service *Service) New(identifier uint64) (interfaces_fuse.DirectoryNode, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("service is closed")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	logger, err := service.loggerFactory.NewLogger("Directory Node")
	if err != nil {
		service.logger.Error("failed to create logger for new directory node", err)
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("failed to create directory node service", err)
		return nil, err
	}

	streamableNodeService, err := service.streamableNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("failed to create streamable node service", err)
		return nil, err
	}

	fileNodeService, err := service.fileNodeServiceFactory.New(service.client)
	if err != nil {
		service.logger.Error("failed to create file node service", err)
		return nil, err
	}

	newNode, err := node.New(service.client, service.loggerFactory, directoryNodeService, streamableNodeService, fileNodeService, logger, identifier)
	if err != nil {
		service.logger.Error("failed to create new directory node", err)
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
