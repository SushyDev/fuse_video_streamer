package factory

import (
	filesystem_client_interfaces "fuse_video_steamer/filesystem/client/interfaces"
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/streamable/handle/service"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
)

type Factory struct {}

var _ interfaces.StreamableHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.StreamableNode, client filesystem_client_interfaces.Client) (interfaces.StreamableHandleService, error) {
	service := service.New(node, client)

	return service, nil
}

