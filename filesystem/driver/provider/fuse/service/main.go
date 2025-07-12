package service

import (
	interfaces "fuse_video_streamer/filesystem/interfaces"
	filesystem_server_provider_fuse "fuse_video_streamer/filesystem/driver/provider/fuse"
	filesystem_server_provider_fuse_filesystem "fuse_video_streamer/filesystem/driver/provider/fuse/filesystem"
	filesystem_server_provider_fuse_root_node_service_factory "fuse_video_streamer/filesystem/driver/provider/fuse/filesystem/root/node/service/factory"
	filesystem_server_provider_fuse_interfaces "fuse_video_streamer/filesystem/driver/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse"
)

type FuseService struct {
	rootNodeServiceFactory filesystem_server_provider_fuse_interfaces.RootNodeServiceFactory
}

var _ interfaces.FileSystemServerService = &FuseService{}

func New() *FuseService {
	rootNodeServiceFactory := filesystem_server_provider_fuse_root_node_service_factory.New()

	return &FuseService{
		rootNodeServiceFactory: rootNodeServiceFactory,
	}
}

func (service *FuseService) New(mountpoint string, volumeName string) interfaces.FileSystemServer {
	logger, err := logger.NewLogger("Fuse")
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
		logger.Fatal("Failed to mount filesystem", err)
	}

	logger.Info("Successfully created connection")

	rootNodeService, err := service.rootNodeServiceFactory.New()
	if err != nil {
		logger.Fatal("Failed to create root node service", err)
	}

	fileSystem := filesystem_server_provider_fuse_filesystem.New(rootNodeService)

	return filesystem_server_provider_fuse.New(mountpoint, connection, fileSystem, logger)
}
