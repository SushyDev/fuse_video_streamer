package service

import (
	"sync/atomic"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	node_root "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node"
)

type Service struct {
	fileSystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_fuse.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory

	loggerFactory interfaces_logger.LoggerFactory

	tree interfaces_fuse.Tree

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_fuse.RootNodeService = &Service{}

func New(
	filesystemClientRepository interfaces_filesystem_client.ClientRepository,

	directoryNodeServiceFactory interfaces_fuse.DirectoryNodeServiceFactory,
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory,

	loggerFactory interfaces_logger.LoggerFactory,

	tree interfaces_fuse.Tree,

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

func (service *Service) New() (interfaces_fuse.RootNode, error) {
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
	directoryNodeService, err := service.directoryNodeServiceFactory.New(nil, service.tree)
	if err != nil {
		service.logger.Error("failed to create Directory Node Service for Root Node", err)
		return nil, err
	}

	identifier := service.tree.GetNextIdentifier()

	rootNode, err := node_root.New(
		service.fileSystemClientRepository,
		service.directoryNodeServiceFactory,
		service.directoryHandleServiceFactory,
		service.loggerFactory,
		directoryNodeService,
		logger,
		service.tree,
		identifier,
	)
	if err != nil {
		service.logger.Error("failed to create Root Node", err)
		return nil, err
	}

	err = service.tree.RegisterNodeOnIdentifier(identifier, rootNode)
	if err != nil {
		service.logger.Error("failed to register Root Node in tree", err)
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
