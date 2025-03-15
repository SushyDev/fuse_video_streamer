package factory

import (
	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/streamable/handle/service"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {}

var _ interfaces.StreamableHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(node interfaces.StreamableNode, apiClient api.FileSystemServiceClient) (interfaces.StreamableHandleService, error) {
	service := service.New(node, apiClient)

	return service, nil
}

