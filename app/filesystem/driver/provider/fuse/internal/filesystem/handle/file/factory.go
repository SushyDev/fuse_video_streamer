package file

import (
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.FileHandleServiceFactory = &Factory{}

func NewFactory(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewService() interfaces_handle.FileHandleService {
	return NewService(factory.loggerFactory)
}
