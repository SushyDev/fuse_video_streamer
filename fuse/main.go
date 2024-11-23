package fuse

import (
	"fuse_video_steamer/fuse/filesystem"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

type Fuse struct {
	fileSystem *vfs.FileSystem
	server     *fs.Server
	logger     *zap.SugaredLogger
}

func New(mountpoint string, fileSystem *vfs.FileSystem) *Fuse {
	fuseLogger, err := logger.GetLogger(logger.FuseLogPath)
	if err != nil {
		panic(err)
	}

	fuseLogger.Info("Creating FUSE instance")

	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName("fuse_video_steamer"),
		fuse.Subtype("fuse_video_steamer"),
		fuse.FSName("fuse_video_steamer"),

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
		server:     fs.New(connection, nil),
		fileSystem: fileSystem,
		logger:     fuseLogger,
	}
}

func (fuse *Fuse) Serve() {
	fuse.logger.Info("Serving FUSE filesystem")

	fileSystem := filesystem.New(fuse.fileSystem)

	err := fuse.server.Serve(fileSystem)
	if err != nil {
		fuse.logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}
}

func (fuse *Fuse) GetVirtualFileSystem() *vfs.FileSystem {
	return fuse.fileSystem
}

func (fuse *Fuse) GetServer() *fs.Server {
	return fuse.server
}

func getNodeID(ID uint64) fuse.NodeID {
	return fuse.NodeID(ID)
}
