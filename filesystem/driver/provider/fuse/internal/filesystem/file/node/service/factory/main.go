package factory

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/node/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type Factory struct {}

var _ interfaces.FileNodeServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(client filesystem_client_interfaces.Client) (interfaces.FileNodeService, error) {
	return service.New(client)
}
