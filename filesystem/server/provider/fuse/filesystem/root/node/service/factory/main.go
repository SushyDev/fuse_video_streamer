package factory

import (
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/root/node/service"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/directory/node/service/factory"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
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
