package directory

import (
	"fuse_video_streamer/config"
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	node_file "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/file"
	node_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/streamable"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
)

type Factory struct {
	config        *config.Config
	loggerFactory interfaces_logger.LoggerFactory
	registry      registry.Registry
}

var _ interfaces_node.DirectoryNodeServiceFactory = &Factory{}

func NewFactory(config *config.Config, loggerFactory interfaces_logger.LoggerFactory, registry registry.Registry) *Factory {
	return &Factory{
		config:        config,
		loggerFactory: loggerFactory,
		registry:      registry,
	}
}

func (factory *Factory) NewService(client interfaces_filesystem_client.Client, tree interfaces_node.Tree) (interfaces_node.DirectoryNodeService, error) {
	// Use the injected registry, or create a fresh one per client service when none provided.
	serviceRegistry := factory.registry
	if serviceRegistry == nil {
		serviceRegistry = registry.New()
	}

	directoryNodeServiceFactory := NewFactory(factory.config, factory.loggerFactory, serviceRegistry)
	streamableNodeServiceFactory := node_streamable.NewFactory(factory.config, factory.loggerFactory, serviceRegistry)
	fileNodeServiceFactory := node_file.NewFactory(factory.loggerFactory, serviceRegistry)

	logger, err := factory.loggerFactory.NewLogger("Directory Node Service")
	if err != nil {
		return nil, err
	}

	return NewService(
		client,
		directoryNodeServiceFactory,
		streamableNodeServiceFactory,
		fileNodeServiceFactory,
		factory.loggerFactory,
		logger,
		tree,
		serviceRegistry,
	)
}
