package fuse

import (
	"fuse_video_steamer/config"
	"fuse_video_steamer/fuse/filesystem"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

type Fuse struct {
	server *fs.Server
	logger *zap.SugaredLogger
}

func New(mountpoint string) *Fuse {
	fuseLogger, err := logger.GetLogger(logger.FuseLogPath)
	if err != nil {
		panic(err)
	}

	fuseLogger.Info("Creating FUSE instance")

	volumeName := config.GetVolumeName()

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
		fuseLogger.Fatalf("Failed to create FUSE mount: %v", err)
	}

	return &Fuse{
		server: fs.New(connection, nil),
		logger: fuseLogger,
	}
}

func (fuse *Fuse) Serve() {
	fuse.logger.Info("Serving FUSE filesystem")

	fileSystem := filesystem.New()

	err := fuse.server.Serve(fileSystem)
	if err != nil {
		fuse.logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}
}

func (fuse *Fuse) GetServer() *fs.Server {
	return fuse.server
}

func getNodeID(ID uint64) fuse.NodeID {
	return fuse.NodeID(ID)
}
