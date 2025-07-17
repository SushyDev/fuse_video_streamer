package service

import (
	"fmt"
	"sync"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Service struct {
	client interfaces_filesystem_client.Client

	logger   interfaces_logger.Logger
	registry *registry.Registry

	loggerFactory                  interfaces_logger.LoggerFactory
	streamableHandleServiceFactory interfaces_fuse.StreamableHandleServiceFactory

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.StreamableNodeService = &Service{}

func New(
	client interfaces_filesystem_client.Client,
	logger interfaces_logger.Logger,
	loggerFactory interfaces_logger.LoggerFactory,
	streamableHandleServiceFactory interfaces_fuse.StreamableHandleServiceFactory,
) (interfaces_fuse.StreamableNodeService, error) {
	registry := registry.GetInstance(client)

	service := &Service{
		client: client,
		logger: logger,

		loggerFactory:                  loggerFactory,
		streamableHandleServiceFactory: streamableHandleServiceFactory,

		registry: registry,
	}

	return service, nil
}

func (service *Service) New(identifier uint64) (interfaces_fuse.StreamableNode, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("service is closed")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	fileSystem := service.client.GetFileSystem()

	size, err := fileSystem.GetFileInfo(identifier)
	if err != nil {
		message := fmt.Sprintf("failed to get video size for %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	logger, err := service.loggerFactory.NewLogger("Root Node")
	if err != nil {
		message := fmt.Sprintf("failed to create logger for streamable node with identifier %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	newNode, err := node.New(service.client, logger, service.streamableHandleServiceFactory, identifier, size)
	if err != nil {
		message := fmt.Sprintf("failed to create new streamable node with identifier %d", identifier)
		service.logger.Error(message, err)
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
