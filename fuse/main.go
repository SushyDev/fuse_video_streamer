package fuse

import (
	"context"
	"fuse_video_steamer/fuse/filesystem"
	"fuse_video_steamer/logger"
	"sync"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Fuse struct {
	mountpoint string
	connection *fuse.Conn
	logger     *logger.Logger
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

	err := server.Serve(fileSystem)
	if err != nil {
		instance.logger.Fatal("Failed to serve filesystem", err)
	}

	wg.Wait()

	instance.logger.Info("Filesystem shutdown")
}

func (instance *Fuse) Close() error {
	instance.logger.Info("Shutting down filesystem")

	err := fuse.Unmount(instance.mountpoint)
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
