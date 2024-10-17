package vfs

import (
	"debrid_drive/logger"
	"strings"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type AddFileRequest struct {
	Path     string
	VideoUrl string
	Size     int64
}

func Mount(mountpoint string, request chan AddFileRequest) {
	channel, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName("debrid_drive"),
		fuse.Subtype("debrid_drive"),
		fuse.FSName("debrid_drive"),

        fuse.NoAppleDouble(),
        fuse.NoBrowse(),

        fuse.LocalVolume(),
        // fuse.AsyncRead(),
	)
	if err != nil {
		logger.Logger.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer channel.Close()
	defer logger.Logger.Info("FUSE filesystem unmounted")
	logger.Logger.Info("Mounted FUSE filesystem")

	fileSystem := NewFileSystem()

    go serveChannel(channel, fileSystem)

	for request := range request {
		handleAddFileRequest(request, fileSystem)
	}

	<-channel.Ready
}

func handleAddFileRequest(request AddFileRequest, fileSystem *FileSystem) {
	components := strings.Split(request.Path, "/")

	name := components[len(components)-1]

	directory, err := fileSystem.FindDirectory(components[0])
	if err != nil {
		directory, err = fileSystem.AddDirectory(fileSystem.directory, components[0])
		if err != nil {
			logger.Logger.Errorf("Error adding directory %s: %v", components[0], err)
			return
		}
	}

	fileSystem.AddFile(directory, name, request.VideoUrl, request.Size)
}

func serveChannel(channel *fuse.Conn, fileSystem *FileSystem) {
    logger.Logger.Info("Serving FUSE filesystem")

    err := fs.Serve(channel, fileSystem)
    if err != nil {
        logger.Logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
    }
}
