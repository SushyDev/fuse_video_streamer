package file

import (
	"fmt"
	"sync"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"

	api "github.com/sushydev/stream_mount_api"
)

type Service struct {
	client                   interfaces_filesystem_client.Client
	fileHandleServiceFactory interfaces_handle.FileHandleServiceFactory
	loggerFactory            interfaces_logger.LoggerFactory
	logger                   interfaces_logger.Logger
	tree                     interfaces_node.Tree

	registry *registry.Registry

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_node.FileNodeService = &Service{}

var clients = []api.FileSystemServiceClient{}

func NewService(
	client interfaces_filesystem_client.Client,
	fileHandleFactory interfaces_handle.FileHandleServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	logger interfaces_logger.Logger,
	tree interfaces_node.Tree,
) (interfaces_node.FileNodeService, error) {
	registry := registry.GetInstance(client)

	return &Service{
		client:                   client,
		fileHandleServiceFactory: fileHandleFactory,
		loggerFactory:            loggerFactory,
		logger:                   logger,
		tree:                     tree,

		registry: registry,
	}, nil
}

func (service *Service) NewNode(parentDirectoryNode interfaces_node.DirectoryNode, remoteIdentifier uint64) (interfaces_node.FileNode, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("service is closed")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	metrics := metrics.NewFileNodeMetrics(remoteIdentifier)
	fileSystem := service.client.GetFileSystem()
	fileHandleService := service.fileHandleServiceFactory.NewService()

	fileNodeLogger, err := service.loggerFactory.NewLogger("File Node")
	if err != nil {
		service.logger.Error("Failed to create logger for new file node", err)
		return nil, err
	}

	size, mode, err := fileSystem.GetFileInfo(remoteIdentifier)
	if err != nil {
		message := fmt.Sprintf("failed to get file info for %d", remoteIdentifier)
		service.logger.Error(message, err)
		return nil, err
	}

	identifier := service.tree.GetNextIdentifier()

	newNode := NewNode(service.client, service.loggerFactory, fileHandleService, metrics, fileNodeLogger, identifier, remoteIdentifier, size, mode)

	service.tree.RegisterNode(newNode)

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
