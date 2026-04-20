package main

import (
	"context"
	"fmt"
	"os"

	config_model "fuse_video_streamer/config"
	"fuse_video_streamer/flags"
	"fuse_video_streamer/healthcheck"

	filesystem_server_provider_fuse "fuse_video_streamer/filesystem/driver/provider/fuse"
	filesystem_server_service "fuse_video_streamer/filesystem/driver/service"
	zap_logger "fuse_video_streamer/logger/drivers/zap"

	"net/http"
	_ "net/http/pprof"
	"os/signal"
	"syscall"
)

func main() {
	// go debug()

	// Handle health check flag
	if *flags.GetHealthCheck() {
		performHealthCheck()
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := config_model.Get()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mountpoint := config.GetMountPoint()
	volumeName := config.GetVolumeName()

	zapLoggerFactory := zap_logger.NewFactory()

	fuseService, err := filesystem_server_provider_fuse.New(config, zapLoggerFactory)
	if err != nil {
		return fmt.Errorf("failed to create fuse provider: %w", err)
	}

	fileSystem, err := filesystem_server_service.New(mountpoint, volumeName, fuseService)
	if err != nil {
		return fmt.Errorf("failed to create filesystem service: %w", err)
	}
	defer fileSystem.Close()

	// Channel to receive Serve() completion or error
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- fileSystem.Serve()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go waitForExit(cancel)

	// Wait for either signal (ctx.Done) or Serve() to complete
	select {
	case <-ctx.Done():
		// Signal received, Close() will be called by defer
		// Wait for Serve() to finish gracefully
		<-serveDone
		return nil
	case err := <-serveDone:
		// Serve() completed on its own (shouldn't normally happen)
		if err != nil {
			return fmt.Errorf("filesystem serve error: %w", err)
		}
		return nil
	}
}

func performHealthCheck() {
	config, err := config_model.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
		os.Exit(1)
	}

	mountpoint := config.GetMountPoint()
	healthChecker := healthcheck.New(mountpoint)

	err = healthChecker.Check()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
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
