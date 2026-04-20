package root

import (
	"fuse_video_streamer/config"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	handle_directory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/directory"
	node_directory "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/directory"
)

type ServiceFactory struct {
	filesystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory
	directoryNodeServiceFactory   interfaces_node.DirectoryNodeServiceFactory

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_node.RootNodeServiceFactory = &ServiceFactory{}

func NewFactory(
	filesystemClientRepository interfaces_filesystem_client.ClientRepository,
	config *config.Config,
	loggerFactory interfaces_logger.LoggerFactory,
) (*ServiceFactory, error) {
	directoryHandleServiceFactory := handle_directory.NewFactory(loggerFactory)
	// Pass nil registry: directory factory creates a fresh registry per NewService call.
	directoryNodeServiceFactory := node_directory.NewFactory(config, loggerFactory, nil)

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
