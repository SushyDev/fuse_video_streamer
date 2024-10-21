package fuse

import (
	"debrid_drive/logger"
	"debrid_drive/vfs"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

// todo argument for vfs pointer
func Mount(mountpoint string, virtualFileSystem *vfs.FileSystem) *FuseFileSystem {
	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName("debrid_drive"),
		fuse.Subtype("debrid_drive"),
		fuse.FSName("debrid_drive"),

		fuse.LocalVolume(),
		fuse.AllowOther(),

		fuse.NoAppleDouble(),
		fuse.NoBrowse(),
	)
	if err != nil {
		logger.Logger.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}

	logger.Logger.Info("Mounted FUSE filesystem")

	fileSystem := NewFileSystem(connection, virtualFileSystem)

	go serve(fileSystem)

	return fileSystem
}

func serve(fileSystem *FuseFileSystem) {
	logger.Logger.Info("Serving FUSE filesystem")

	err := fs.Serve(fileSystem.connection, fileSystem)
	if err != nil {
		logger.Logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}
}
