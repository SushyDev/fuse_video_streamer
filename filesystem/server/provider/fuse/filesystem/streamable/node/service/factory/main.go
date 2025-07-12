package factory

import (
	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/filesystem/server/provider/fuse/filesystem/streamable/node/service"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"
)

type Factory struct {}

var _ interfaces.StreamableNodeServiceFactory = &Factory{}

func New() *Factory {
	return &Factory{}
}

func (factory *Factory) New(client filesystem_client_interfaces.Client) (interfaces.StreamableNodeService, error) {
	logger, err := logger.NewLogger("File Node Service")
	if err != nil {
		return nil, err
	}

	return service.New(client, logger)
}
