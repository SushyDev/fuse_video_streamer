package service

import (
	"sync/atomic"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/logger"
)

type Service struct {
	closed atomic.Bool
}

var _ interfaces.DirectoryHandleService = &Service{}

func New() *Service {
	return &Service{}
}

func (service *Service) New(node interfaces.DirectoryNode) (interfaces.DirectoryHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("Directory Handle")
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
