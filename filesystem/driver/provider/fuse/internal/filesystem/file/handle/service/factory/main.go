package factory

import (
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/file/handle/service"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
)

type Factory struct {}

var _ interfaces.FileHandleServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New() interfaces.FileHandleService {
	return service.New()
}

