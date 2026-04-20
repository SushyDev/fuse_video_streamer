// Package main implements a FUSE filesystem that streams video files from remote servers.
//
// The application mounts a FUSE filesystem at the configured mountpoint and proxies
// filesystem operations to remote gRPC file servers. Video files are streamed on-demand
// using HTTP range requests with ring buffer optimization.
//
// Graceful Shutdown:
//   - The application handles SIGINT, SIGTERM, and SIGQUIT signals
//   - On signal receipt, Close() is called on the filesystem to unmount cleanly
//   - The main goroutine waits for Serve() to complete before exiting
//   - All active streams and file handles are closed during shutdown
//
// Usage:
//   fuse_video_streamer              # Start the FUSE server
//   fuse_video_streamer --health     # Check if mount is healthy
package main

import (
	"context"
	"fmt"
	"os"

	config_model "fuse_video_streamer/config"
	"fuse_video_streamer/flags"
	"fuse_video_streamer/healthcheck"

	filesystem_server_provider_fuse "fuse_video_streamer/filesystem/driver/provider/fuse"
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

	mountpoint := config.MountPoint
	volumeName := config.VolumeName

	zapLoggerFactory := zap_logger.NewFactory()

	fuseService, err := filesystem_server_provider_fuse.New(config, zapLoggerFactory)
	if err != nil {
		return fmt.Errorf("failed to create fuse provider: %w", err)
	}

	fileSystem, err := fuseService.New(mountpoint, volumeName)
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

	mountpoint := config.MountPoint
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
