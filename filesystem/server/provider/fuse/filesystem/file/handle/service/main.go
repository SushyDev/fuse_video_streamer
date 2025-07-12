package service

import (
	"sync/atomic"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/file/handle"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"
)

type Service struct {
	node   interfaces.FileNode
	client filesystem_client_interfaces.Client

	closed atomic.Bool
}

var _ interfaces.FileHandleService = &Service{}

func New(node interfaces.FileNode, client filesystem_client_interfaces.Client) *Service {
	return &Service{
		node:   node,
		client: client,
	}
}

func (service *Service) New() (interfaces.FileHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("File Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(service.node, logger), nil
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
