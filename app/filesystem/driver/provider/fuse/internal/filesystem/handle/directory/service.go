package directory

import (
	"fmt"
	"sync/atomic"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

type Service struct {
	closed atomic.Bool

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_handle.DirectoryHandleService = &Service{}

func NewService(loggerFactory interfaces_logger.LoggerFactory) *Service {
	return &Service{
		loggerFactory: loggerFactory,
	}
}

func (service *Service) NewHandle(node interfaces_node.DirectoryNode) (interfaces_handle.DirectoryHandle, error) {
	if service.IsClosed() {
		return nil, fmt.Errorf("directory handle service is closed")
	}

	logger, err := service.loggerFactory.NewLogger("Directory Handle")
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
