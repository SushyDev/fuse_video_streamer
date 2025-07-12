package main

import (
	"context"
	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/interfaces"
	filesystem_server_provider_fuse_service "fuse_video_streamer/filesystem/driver/provider/fuse/service"
	filesystem_server_service "fuse_video_streamer/filesystem/driver/service"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// go debug()
	
	config.Validate()

	mountpoint := config.GetMountPoint()
	volumeName := config.GetVolumeName()

	var fileSystemProvider interfaces.FileSystemServerService
	fileSystemProvider = filesystem_server_provider_fuse_service.New()

	fileSystem := filesystem_server_service.New(mountpoint, volumeName, fileSystemProvider)

	go fileSystem.Serve()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go waitForExit(cancel)

	<-ctx.Done()

	fileSystem.Close()
}

func waitForExit(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-signals

	cancel()
}

func debug() {
		fmt.Println("Pprof server started on localhost:6060")
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
}

