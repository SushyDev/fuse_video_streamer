package root

import (
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

type Service struct {
	fileSystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_node.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory

	loggerFactory interfaces_logger.LoggerFactory

	tree interfaces_node.Tree

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_node.RootNodeService = &Service{}

func NewService(
	filesystemClientRepository interfaces_filesystem_client.ClientRepository,

	directoryNodeServiceFactory interfaces_node.DirectoryNodeServiceFactory,
	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory,

	loggerFactory interfaces_logger.LoggerFactory,

	tree interfaces_node.Tree,

	logger interfaces_logger.Logger,
) *Service {
	return &Service{
		fileSystemClientRepository: filesystemClientRepository,

		directoryNodeServiceFactory:   directoryNodeServiceFactory,
		directoryHandleServiceFactory: directoryHandleServiceFactory,

		loggerFactory: loggerFactory,

		tree: tree,

		logger: logger,
	}
}

func (service *Service) NewNode() (interfaces_node.RootNode, error) {
	if service.IsClosed() {
		service.logger.Error("root Node Service is closed, cannot create new root node", nil)
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("Root Node")
	if err != nil {
		service.logger.Error("failed to create logger for Root Node", err)
		return nil, err
	}

	// Root node does not have a client since it lists all clients from the client repository
	directoryNodeService, err := service.directoryNodeServiceFactory.NewService(nil, service.tree)
	if err != nil {
		service.logger.Error("failed to create Directory Node Service for Root Node", err)
		return nil, err
	}

	rootNode, err := NewNode(
		service.fileSystemClientRepository,
		service.directoryNodeServiceFactory,
		service.directoryHandleServiceFactory,
		service.loggerFactory,
		directoryNodeService,
		logger,
		service.tree,
	)
	if err != nil {
		service.logger.Error("failed to create Root Node", err)
		return nil, err
	}

	return rootNode, nil
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
