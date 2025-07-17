package service

import (
	"fmt"
	"sync"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	file_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"

	api "github.com/sushydev/stream_mount_api"
)

type Service struct {
	client interfaces_filesystem_client.Client
	logger interfaces_logger.Logger

	loggerFactory            interfaces_logger.LoggerFactory
	fileHandleServiceFactory interfaces_fuse.FileHandleServiceFactory

	registry *registry.Registry

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.FileNodeService = &Service{}

var clients = []api.FileSystemServiceClient{}

func New(
	client interfaces_filesystem_client.Client,
	logger interfaces_logger.Logger,
	loggerFactory interfaces_logger.LoggerFactory,
	fileHandleFactory interfaces_fuse.FileHandleServiceFactory,
) (interfaces_fuse.FileNodeService, error) {
	registry := registry.GetInstance(client)

	return &Service{
		client: client,
		logger: logger,

		loggerFactory:            loggerFactory,
		fileHandleServiceFactory: fileHandleFactory,

		registry: registry,
	}, nil
}

func (service *Service) New(identifier uint64) (interfaces_fuse.FileNode, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("service is closed")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	metrics := metrics.NewFileNodeMetrics(identifier)
	fileSystem := service.client.GetFileSystem()
	fileHandleService := service.fileHandleServiceFactory.New()

	fileNodeLogger, err := service.loggerFactory.NewLogger("File Node")
	if err != nil {
		service.logger.Error("Failed to create logger for new file node", err)
		return nil, err
	}

	size, err := fileSystem.GetFileInfo(identifier)
	if err != nil {
		message := fmt.Sprintf("Failed to get video size for %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	newNode := file_node.New(service.client, service.loggerFactory, fileHandleService, metrics, fileNodeLogger, identifier, size)

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
