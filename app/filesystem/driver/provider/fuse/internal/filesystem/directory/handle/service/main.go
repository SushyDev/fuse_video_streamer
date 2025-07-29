package service

import (
	"fmt"
	"sync/atomic"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	directory_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle"
)

type Service struct {
	closed atomic.Bool

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.DirectoryHandleService = &Service{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Service {
	return &Service{
		loggerFactory: loggerFactory,
	}
}

func (service *Service) New(node interfaces_fuse.DirectoryNode) (interfaces_fuse.DirectoryHandle, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("directory handle service is closed")
	}

	logger, err := service.loggerFactory.NewLogger("Directory Handle")
	if err != nil {
		return nil, err
	}

	return directory_handle.New(node, logger), nil
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
