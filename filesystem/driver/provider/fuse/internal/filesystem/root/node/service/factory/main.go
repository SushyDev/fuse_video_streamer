package factory

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/node/service/factory"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type ServiceFactory struct {
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory
}

var _ interfaces.RootNodeServiceFactory = &ServiceFactory{}

func New() *ServiceFactory {
	directoryNodeServiceFactory := factory.New()

	return &ServiceFactory{
		directoryNodeServiceFactory: directoryNodeServiceFactory,
	}
}

func (factory *ServiceFactory) New() (interfaces.RootNodeService, error) {
	return service.New(factory.directoryNodeServiceFactory), nil
}
