package factory

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
)

type Factory struct {}

var _ interfaces.DirectoryHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.DirectoryNode, client filesystem_client_interfaces.Client) (interfaces.DirectoryHandleService, error) {
	return service.New(node, client), nil
}

