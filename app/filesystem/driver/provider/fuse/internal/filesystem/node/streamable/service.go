package streamable

import (
	"fmt"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Service struct {
	client interfaces_filesystem_client.Client
	logger interfaces_logger.Logger

	streamableHandleServiceFactory handle.StreamableHandleServiceFactory
	loggerFactory                  interfaces_logger.LoggerFactory
	tree                           node.Tree

	registry *registry.Registry

	closed atomic.Bool
}

var _ node.StreamableNodeService = &Service{}

func NewService(
	client interfaces_filesystem_client.Client,
	streamableHandleServiceFactory handle.StreamableHandleServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	logger interfaces_logger.Logger,
	tree node.Tree,
) (node.StreamableNodeService, error) {
	registry := registry.GetInstance(client)

	service := &Service{
		client:                         client,
		streamableHandleServiceFactory: streamableHandleServiceFactory,
		loggerFactory:                  loggerFactory,
		logger:                         logger,
		tree:                           tree,

		registry: registry,
	}

	return service, nil
}

func (service *Service) NewNode(parentDirectoryNode node.DirectoryNode, remoteIdentifier uint64) (node.StreamableNode, error) {
	if service.IsClosed() {
		service.logger.Warn("Attempted to create a new Streamable Node after service was closed")
		return nil, fmt.Errorf("service is closed")
	}

	fileSystem := service.client.GetFileSystem()

	size, err := fileSystem.GetFileInfo(remoteIdentifier)
	if err != nil {
		message := fmt.Sprintf("failed to get video size for %d", remoteIdentifier)
		service.logger.Error(message, err)
		return nil, err
	}

	logger, err := service.loggerFactory.NewLogger("Streamable Node")
	if err != nil {
		message := fmt.Sprintf("failed to create logger for streamable node with identifier %d", remoteIdentifier)
		service.logger.Error(message, err)
		return nil, err
	}

	identifier := service.tree.GetNextIdentifier()

	newNode, err := NewNode(service.client, service.streamableHandleServiceFactory, logger, identifier, remoteIdentifier, size)
	if err != nil {
		message := fmt.Sprintf("failed to create new streamable node with identifier %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

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
