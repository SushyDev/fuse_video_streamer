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

	mountpoint := config.GetMountPoint()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handleSignals(cancel)

	fuseInstance := fuse.New(mountpoint)

	fuseInstance.Serve(ctx)
}

func handleSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	<-signals

	cancel()
}
