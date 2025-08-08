package file

import (
	"sync/atomic"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

type Service struct {
	loggerFactory interfaces_logger.LoggerFactory

	closed atomic.Bool
}

var _ interfaces_handle.FileHandleService = &Service{}

func NewService(loggerFactory interfaces_logger.LoggerFactory) *Service {
	return &Service{
		loggerFactory: loggerFactory,
	}
}

func (service *Service) NewHandle(node interfaces_node.FileNode) (interfaces_handle.FileHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := service.loggerFactory.NewLogger("File Handle")
	if err != nil {
		return nil, err
	}

	return NewHandle(node, logger), nil
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
