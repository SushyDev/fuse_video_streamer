package factory

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/filesystem/file/handle/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/interfaces"
)

type Factory struct {}

var _ interfaces.FileHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.FileNode, client filesystem_client_interfaces.Client) (interfaces.FileHandleService, error) {
	service := service.New(node, client)

	return service, nil
}

