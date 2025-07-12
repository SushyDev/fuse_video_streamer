package service

import (
	"sync/atomic"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/directory/handle"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"
)

type Service struct {
	node interfaces.DirectoryNode
	client filesystem_client_interfaces.Client

	closed atomic.Bool
}

var _ interfaces.DirectoryHandleService = &Service{}

func New(node interfaces.DirectoryNode, client filesystem_client_interfaces.Client) *Service {
	return &Service{
		node: node,
		client: client,
	}
}

func (service *Service) New() (interfaces.DirectoryHandle, error) {
	if service.IsClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("Directory Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(service.client, service.node, logger), nil
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
