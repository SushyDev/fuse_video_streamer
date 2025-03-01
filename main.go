package main

import (
	"context"
	"fuse_video_steamer/config"
	fuse_service "fuse_video_steamer/fuse/service"
	"fuse_video_steamer/filesystem"
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

	fileSystem := filesystem.New(fuse_service.New(), mountpoint, volumeName)

	fileSystem.Serve(ctx)
}

func waitForExit(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-signals

	cancel()
}
