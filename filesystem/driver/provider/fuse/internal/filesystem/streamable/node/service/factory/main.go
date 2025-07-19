package factory

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_streamable_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/handle/service/factory"

	service_streamable_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/streamable/node/service"
)

type Factory struct {
	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_fuse.StreamableNodeServiceFactory = &Factory{}

func New(loggerFactory interfaces_logger.LoggerFactory) *Factory {
	return &Factory{
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) New(client interfaces_filesystem_client.Client, tree interfaces_fuse.Tree) (interfaces_fuse.StreamableNodeService, error) {
	streamableNodeService, err := factory.loggerFactory.NewLogger("Streamable Node Service")
	if err != nil {
		return nil, err
	}

	streamableHandleServiceFactory := factory_streamable_handle.New(factory.loggerFactory)

	return service_streamable_handle.New(client, streamableHandleServiceFactory, factory.loggerFactory, streamableNodeService, tree)
}
