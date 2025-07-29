package factory

import (
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	service_directory_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle/service"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.DirectoryHandleServiceFactory = &Factory{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) New() interfaces_fuse.DirectoryHandleService {
	return service_directory_handle.New(factory.loggerFactory)
}
