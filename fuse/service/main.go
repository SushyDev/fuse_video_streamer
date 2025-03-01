package service

import (
	"fuse_video_steamer/logger"
	fvs_fuse "fuse_video_steamer/fuse"
	fvs_fuse_filesystem "fuse_video_steamer/fuse/filesystem"
	filesystem_interfaces "fuse_video_steamer/filesystem/interfaces"

	"github.com/anacrolix/fuse"
)

type FuseService struct {}

var _ filesystem_interfaces.FileSystemService = &FuseService{}

func New() *FuseService {
	return &FuseService{}
}

func (service *FuseService) New(mountpoint string, volumeName string) filesystem_interfaces.FileSystem {
	logger, err := logger.NewLogger("Fuse")
	if err != nil {
		panic(err)
	}

	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName(volumeName),
		fuse.Subtype(volumeName),
		fuse.FSName(volumeName),

		fuse.LocalVolume(),
		fuse.AllowOther(),
		fuse.AllowSUID(),

		fuse.NoAppleDouble(),
		fuse.NoBrowse(),
	)

	if err != nil {
		logger.Fatal("Failed to mount filesystem", err)
	}

	logger.Info("Successfully created connection")

	fileSystem := fvs_fuse_filesystem.New()

	return fvs_fuse.New(mountpoint, connection, fileSystem, logger)
}
