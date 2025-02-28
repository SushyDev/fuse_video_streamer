package main

import (
	"context"
	"fuse_video_steamer/config"
	"fuse_video_steamer/fuse"
	"fuse_video_steamer/grafana_logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config.Validate()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go captureExitSignals(cancel)

	go grafana_logger.Record()

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
