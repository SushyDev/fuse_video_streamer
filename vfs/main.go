package vfs

import (
	"debrid_drive/logger"
	"strings"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

type AddFileRequest struct {
	Path     string
	VideoUrl string
	Size     int64
}

func Mount(mountpoint string, done chan bool, request chan AddFileRequest) {
	channel, err := fuse.Mount(mountpoint)
	if err != nil {
		logger.Logger.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer channel.Close()
	logger.Logger.Info("Mounted FUSE filesystem")

	fileSystem := NewFileSystem()

	go serve(channel, fileSystem)
	logger.Logger.Info("Serving FUSE filesystem")

	for request := range request {
		handleAddFileRequest(request, fileSystem)
	}
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

func serve(channel *fuse.Conn, fileSystem *FileSystem) {
	err := fs.Serve(channel, fileSystem)
	if err != nil {
		logger.Logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}
}
