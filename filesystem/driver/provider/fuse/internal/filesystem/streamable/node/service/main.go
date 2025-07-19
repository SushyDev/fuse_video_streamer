package service

import (
	"fmt"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Service struct {
	client                         interfaces_filesystem_client.Client
	streamableHandleServiceFactory interfaces_fuse.StreamableHandleServiceFactory
	loggerFactory                  interfaces_logger.LoggerFactory
	logger                         interfaces_logger.Logger
	tree                           interfaces_fuse.Tree

	registry *registry.Registry

	closed atomic.Bool
}

var _ interfaces_fuse.StreamableNodeService = &Service{}

func New(
	client interfaces_filesystem_client.Client,
	streamableHandleServiceFactory interfaces_fuse.StreamableHandleServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	logger interfaces_logger.Logger,
	tree interfaces_fuse.Tree,
) (interfaces_fuse.StreamableNodeService, error) {
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

func (service *Service) New(parentDirectoryNode interfaces_fuse.DirectoryNode, remoteIdentifier uint64) (interfaces_fuse.StreamableNode, error) {
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

	newNode, err := node.New(service.client, service.streamableHandleServiceFactory, logger, identifier, remoteIdentifier, size)
	if err != nil {
		message := fmt.Sprintf("failed to create new streamable node with identifier %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	service.tree.RegisterNodeOnIdentifier(identifier, newNode)

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
