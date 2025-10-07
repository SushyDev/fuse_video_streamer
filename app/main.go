package main

import (
	"context"
	"fmt"

	config_model "fuse_video_streamer/config"

	filesystem_server_provider_fuse "fuse_video_streamer/filesystem/driver/provider/fuse"
	filesystem_server_service "fuse_video_streamer/filesystem/driver/service"
	zap_logger "fuse_video_streamer/logger/drivers/zap"

	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// go debug()

	config, err := config_model.Get()
	if err != nil {
		panic(err)
	}

	mountpoint := config.GetMountPoint()
	volumeName := config.GetVolumeName()

	zapLoggerFactory := zap_logger.NewFactory()

	fuseService, err := filesystem_server_provider_fuse.New(config, zapLoggerFactory)
	if err != nil {
		panic(err)
	}

	fileSystem, err := filesystem_server_service.New(mountpoint, volumeName, fuseService)
	if err != nil {
		panic(err)
	}

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
