package fuse

import (
	interfaces_filesystem "fuse_video_streamer/filesystem/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	node_root "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/root"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/server"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/tree"

	"github.com/anacrolix/fuse"
)

type FuseService struct {
	rootNodeServiceFactory interfaces_node.RootNodeServiceFactory

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_filesystem.FileSystemServerService = &FuseService{}

func New(loggerFactory interfaces_logger.LoggerFactory) (*FuseService, error) {
	rootNodeServiceFactory, err := node_root.NewFactory(loggerFactory)
	if err != nil {
		return nil, err
	}

	return &FuseService{
		rootNodeServiceFactory: rootNodeServiceFactory,

		loggerFactory: loggerFactory,
	}, nil
}

func (service *FuseService) New(mountpoint string, volumeName string) (interfaces_filesystem.FileSystemServer, error) {
	logger, err := service.loggerFactory.NewLogger("Fuse")
	if err != nil {
		return nil, err
	}

	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName(volumeName),
		fuse.Subtype(volumeName),
		fuse.FSName(volumeName),

		fuse.AllowOther(),
		fuse.LocalVolume(),

		fuse.NoAppleDouble(),
		fuse.NoBrowse(),
	)

	if err != nil {
		return nil, err
	}

	logger.Info("Successfully created connection")

	tree := tree.New()

	rootNodeService, err := service.rootNodeServiceFactory.NewService(tree)
	if err != nil {
		return nil, err
	}

	fileSystemLogger, err := service.loggerFactory.NewLogger("File System")
	if err != nil {
		return nil, err
	}

	fileSystem, err := filesystem.New(tree, rootNodeService, fileSystemLogger)
	if err != nil {
		return nil, err
	}

	return server.New(mountpoint, connection, fileSystem, logger), nil
}
