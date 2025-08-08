package directory

import (
	"fmt"
	"sync"
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	node_abstract "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/abstract"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Service struct {
	client                       interfaces_filesystem_client.Client
	directoryNodeServiceFactory  interfaces_node.DirectoryNodeServiceFactory
	streamableNodeServiceFactory interfaces_node.StreamableNodeServiceFactory
	fileNodeServiceFactory       interfaces_node.FileNodeServiceFactory
	loggerFactory                interfaces_logger.LoggerFactory
	logger                       interfaces_logger.Logger
	tree                         interfaces_node.Tree

	registry *registry.Registry

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_node.DirectoryNodeService = &Service{}

func NewService(
	client interfaces_filesystem_client.Client,
	directoryNodeServiceFactory interfaces_node.DirectoryNodeServiceFactory,
	streamableNodeServiceFactory interfaces_node.StreamableNodeServiceFactory,
	fileNodeServiceFactory interfaces_node.FileNodeServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	logger interfaces_logger.Logger,
	tree interfaces_node.Tree,
) (interfaces_node.DirectoryNodeService, error) {
	registry := registry.GetInstance(client)

	service := &Service{
		client:                       client,
		directoryNodeServiceFactory:  directoryNodeServiceFactory,
		streamableNodeServiceFactory: streamableNodeServiceFactory,
		fileNodeServiceFactory:       fileNodeServiceFactory,
		loggerFactory:                loggerFactory,
		tree:                         tree,
		logger:                       logger,

		registry: registry,
	}

	return service, nil
}

func (service *Service) NewNode(parentDirectoryNode interfaces_node.DirectoryNode, remoteIdentifier uint64) (interfaces_node.DirectoryNode, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("service is closed")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	abstractNode := node_abstract.NewNode(
		service.client,
		service.tree.GetNextIdentifier(),
		remoteIdentifier,
	)

	logger, err := service.loggerFactory.NewLogger("Directory Node")
	if err != nil {
		service.logger.Error("failed to create logger for new directory node", err)
		return nil, err
	}

	directoryNodeService, err := service.directoryNodeServiceFactory.NewService(service.client, service.tree)
	if err != nil {
		service.logger.Error("failed to create directory node service", err)
		return nil, err
	}

	streamableNodeService, err := service.streamableNodeServiceFactory.NewService(service.client, service.tree)
	if err != nil {
		service.logger.Error("failed to create streamable node service", err)
		return nil, err
	}

	fileNodeService, err := service.fileNodeServiceFactory.NewService(service.client, service.tree)
	if err != nil {
		service.logger.Error("failed to create file node service", err)
		return nil, err
	}

	newNode, err := NewNode(abstractNode, logger, service.loggerFactory, directoryNodeService, streamableNodeService, fileNodeService)
	if err != nil {
		service.logger.Error("failed to create new directory node", err)
		return nil, err
	}

	err = service.tree.RegisterNode(newNode)
	if err != nil {
		service.logger.Error("failed to register new directory node in tree", err)
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
