package fuse

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fuse_video_steamer/fuse/filesystem"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Fuse struct {
	mountpoint string
	connection *fuse.Conn
	logger     *logger.Logger

	fileSystem *filesystem.FileSystem
}

func New(mountpoint string, volumeName string) *Fuse {
	logger, err := logger.NewLogger("Fuse")
	if err != nil {
		panic(err)
	}

	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName(volumeName),
		fuse.Subtype(volumeName),
		fuse.FSName(volumeName),

		fuse.LocalVolume(),
		fuse.AllowOther(),
		fuse.AllowSUID(),

		fuse.NoAppleDouble(),
		fuse.NoBrowse(),
	)

	if err != nil {
		logger.Fatal("Failed to mount filesystem", err)
	}

	logger.Info("Successfully created connection")

	return &Fuse{
		mountpoint: mountpoint,
		connection: connection,
		logger:     logger,
	}
}

func (instance *Fuse) Serve(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		<-ctx.Done()
		instance.Close()
		wg.Done()
	}()

	fileSystem := filesystem.New()
	server := fs.New(instance.connection, nil)

	instance.logger.Info("Serving filesystem")

	instance.fileSystem = fileSystem

	err := server.Serve(fileSystem)
	if err != nil {
		instance.logger.Fatal("Failed to serve filesystem", err)
	}

	wg.Wait()

	instance.logger.Info("Filesystem shutdown")
}

func (instance *Fuse) Close() error {
	instance.logger.Info("Shutting down filesystem")

	err := instance.unmount()
	if err != nil {
		instance.logger.Error("Failed to unmount filesystem", err)
	} else {
		instance.logger.Info("Unmounted filesystem")
	}

	if instance.connection != nil {
		err = instance.connection.Close()
		if err != nil {
			instance.logger.Error("Failed to close connection", err)
		} else {
			instance.logger.Info("Closed connection")
		}
	} else {
		instance.logger.Info("Connection already closed")
	}

	instance.logger.Info("Fuse closed")

	return nil
}

func (instance *Fuse) unmount() error {
	instance.fileSystem.Close()

	var unmounted bool
	var err error

	for tries := 0; tries < 10; tries++ {
		err = fuse.Unmount(instance.mountpoint)
		if err == nil {
			unmounted = true
			break
		}

		if strings.HasSuffix(err.Error(), "resource busy") {
			instance.logger.Info("Waiting for filesystem to unmount")
			time.Sleep(1 * time.Second)
			continue
		}

		instance.logger.Error("Failed to unmount filesystem", err)

		break
	}

	if !unmounted {
		return fmt.Errorf("Reached max tries to unmount filesystem. Last error: %v", err)
	}

	return nil
}
