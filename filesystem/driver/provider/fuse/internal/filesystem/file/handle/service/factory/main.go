package factory

import (
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/handle/service"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.FileHandleServiceFactory = &Factory{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) New() interfaces_fuse.FileHandleService {
	return service.New(factory.loggerFactory)
}
