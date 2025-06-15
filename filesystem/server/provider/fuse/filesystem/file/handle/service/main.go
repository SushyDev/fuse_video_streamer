package service

import (
	"context"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/file/handle"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"
)

type Service struct {
	node   interfaces.FileNode
	client filesystem_client_interfaces.Client

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileHandleService = &Service{}

func New(node interfaces.FileNode, client filesystem_client_interfaces.Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		node:   node,
		client: client,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (service *Service) New() (interfaces.FileHandle, error) {
	if service.isClosed() {
		return nil, nil
	}

	logger, err := logger.NewLogger("File Handle")
	if err != nil {
		return nil, err
	}

	return handle.New(service.node, logger), nil
}

func (service *Service) Close() error {
	service.cancel()

	return nil
}

func (service *Service) isClosed() bool {
	select {
	case <-service.ctx.Done():
		return true
	default:
		return false
	}
}
