package directory

import (
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.DirectoryHandleServiceFactory = &Factory{}

func NewFactory(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService() interfaces_handle.DirectoryHandleService {
	return NewService(factory.loggerFactory)
}
