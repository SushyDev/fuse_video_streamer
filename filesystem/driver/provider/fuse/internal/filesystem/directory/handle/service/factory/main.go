package factory

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/directory/handle/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type Factory struct {}

var _ interfaces.DirectoryHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New() interfaces.DirectoryHandleService {
	return service.New()
}
