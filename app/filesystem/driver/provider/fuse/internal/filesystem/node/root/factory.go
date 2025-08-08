package root

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	handle_directory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/directory"
	node_directory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/directory"

	filesystem_client_repository "fuse_video_streamer/filesystem/client/repository"
)

type ServiceFactory struct {
	filesystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory
	directoryNodeServiceFactory   interfaces_node.DirectoryNodeServiceFactory

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_node.RootNodeServiceFactory = &ServiceFactory{}

func NewFactory(loggerFactory interfaces_logger.LoggerFactory) (*ServiceFactory, error) {
	filesystemClientRepositoryLogger, err := loggerFactory.NewLogger("Filesystem Client Repository")
	if err != nil {
		return nil, err
	}

	filesystemClientRepository, err := filesystem_client_repository.New(loggerFactory, filesystemClientRepositoryLogger)
	if err != nil {
		return nil, err
	}

	directoryHandleServiceFactory := handle_directory.NewFactory(loggerFactory)
	directoryNodeServiceFactory := node_directory.NewFactory(loggerFactory)

	return &ServiceFactory{
		filesystemClientRepository: filesystemClientRepository,

		directoryHandleServiceFactory: directoryHandleServiceFactory,
		directoryNodeServiceFactory:   directoryNodeServiceFactory,

		loggerFactory: loggerFactory,
	}, nil
}

func (factory *ServiceFactory) NewService(tree interfaces_node.Tree) (interfaces_node.RootNodeService, error) {
	nodeServiceLogger, err := factory.loggerFactory.NewLogger("Root Node Service")
	if err != nil {
		return nil, err
	}

	return NewService(
		factory.filesystemClientRepository,
		factory.directoryNodeServiceFactory,
		factory.directoryHandleServiceFactory,
		factory.loggerFactory,
		tree,
		nodeServiceLogger,
	), nil
}
