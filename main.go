package main

import (
	"context"
	"fuse_video_steamer/config"
	"fuse_video_steamer/filesystem/interfaces"
	filesystem_server_service "fuse_video_steamer/filesystem/server"
	fuse_service "fuse_video_steamer/filesystem/server/providers/fuse/service"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go waitForExit(cancel)

	mountpoint := config.GetMountPoint()
	volumeName := config.GetVolumeName()

	var fileSystemProvider interfaces.FileSystemServerService
	fileSystemProvider = fuse_service.New()


	fileSystem := filesystem_server_service.New(mountpoint, volumeName, fileSystemProvider)

	fileSystem.Serve(ctx)
}

func waitForExit(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-signals

	cancel()
}
