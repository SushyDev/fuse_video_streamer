package main

import (
	"context"
	"fuse_video_steamer/config"
	"fuse_video_steamer/fuse"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go captureExitSignals(cancel)

	mountpoint := config.GetMountPoint()
	volumeName := config.GetVolumeName()

	fuseInstance := fuse.New(mountpoint, volumeName)

	fuseInstance.Serve(ctx)
}

func captureExitSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals

	cancel()
}
