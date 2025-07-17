package factory

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_directory_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle/service/factory"
	factory_directory_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node/service/factory"

	service_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node/service"

	"fuse_video_streamer/filesystem/client/repository"
)

type ServiceFactory struct {
	filesystemClientRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_fuse.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory
	directoryNodeService          interfaces_fuse.DirectoryNodeService

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.RootNodeServiceFactory = &ServiceFactory{}

func New(loggerFactory interfaces_logger.LoggerFactory) (*ServiceFactory, error) {
	filesystemClientRepositoryLogger, err := loggerFactory.NewLogger("Filesystem Client Repository")
	if err != nil {
		return nil, err
	}

	filesystemClientRepository, err := repository.New(loggerFactory, filesystemClientRepositoryLogger)
	if err != nil {
		return nil, err
	}

	directoryNodeServiceFactory := factory_directory_node.New(loggerFactory)
	directoryHandleServiceFactory := factory_directory_handle.New(loggerFactory)

	return &ServiceFactory{
		filesystemClientRepository: filesystemClientRepository,

		directoryNodeServiceFactory:   directoryNodeServiceFactory,
		directoryHandleServiceFactory: directoryHandleServiceFactory,

		loggerFactory: loggerFactory,
	}, nil
}

func (factory *ServiceFactory) New() (interfaces_fuse.RootNodeService, error) {
	nodeServiceLogger, err := factory.loggerFactory.NewLogger("Root Node Service")
	if err != nil {
		return nil, err
	}

	return service_node.New(
		factory.filesystemClientRepository,
		factory.directoryNodeServiceFactory,
		factory.directoryHandleServiceFactory,
		factory.loggerFactory,
		nodeServiceLogger,
	), nil
}
