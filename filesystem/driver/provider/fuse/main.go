package fuse

import (
	interfaces_fuse_filesystem "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_filesystem "fuse_video_streamer/filesystem/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	factory_root_node_service "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/root/node/service/factory"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/server"

	"github.com/anacrolix/fuse"
)

type FuseService struct {
	rootNodeServiceFactory interfaces_fuse_filesystem.RootNodeServiceFactory

	loggerFactory interfaces_logger.LoggerFactory
}

var _ interfaces_filesystem.FileSystemServerService = &FuseService{}

func New(loggerFactory interfaces_logger.LoggerFactory) *FuseService {
	rootNodeServiceFactory := factory_root_node_service.New(loggerFactory)

	return &FuseService{
		rootNodeServiceFactory: rootNodeServiceFactory,

		loggerFactory: loggerFactory,
	}
}

func (service *FuseService) New(mountpoint string, volumeName string) interfaces_filesystem.FileSystemServer {
	logger, err := service.loggerFactory.NewLogger("Fuse")
	if err != nil {
		panic(err)
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
		logger.Fatal("failed to mount filesystem", err)
	}

	logger.Info("Successfully created connection")

	rootNodeService, err := service.rootNodeServiceFactory.New()
	if err != nil {
		logger.Fatal("failed to create root node service", err)
	}

	fileSystemLogger, err := service.loggerFactory.NewLogger("File System")
	if err != nil {
		logger.Fatal("failed to create file system logger", err)
	}

	fileSystem := filesystem.New(rootNodeService, fileSystemLogger)

	return server.New(mountpoint, connection, fileSystem, logger)
}
