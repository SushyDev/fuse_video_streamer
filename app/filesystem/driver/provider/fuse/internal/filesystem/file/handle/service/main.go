package service

import (
	"sync/atomic"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/handle"
)

type Service struct {
	loggerFactory interfaces_logger.LoggerFactory

	closed atomic.Bool
}

var _ interfaces_fuse.FileHandleService = &Service{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Service {
	return &Service{
		loggerFactory: loggerFactory,
	}
}

func (service *Service) New(node interfaces_fuse.FileNode) (interfaces_fuse.FileHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("File Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(node, logger), nil
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
