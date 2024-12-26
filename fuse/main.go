package fuse

import (
	"context"
	"fuse_video_steamer/config"
	"fuse_video_steamer/fuse/filesystem"
	"fuse_video_steamer/logger"
	"sync"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

type Fuse struct {
	mountpoint string
	connection *fuse.Conn
	logger     *zap.SugaredLogger
}

func New(mountpoint string) *Fuse {
	fuseLogger, err := logger.GetLogger(logger.FuseLogPath)
	if err != nil {
		panic(err)
	}

	volumeName := config.GetVolumeName()

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
        fuseLogger.Fatalf("Failed to create connection: %v", err)
	}

	fuseLogger.Info("Successfully created connection")

	return &Fuse{
		mountpoint: mountpoint,
		connection: connection,
		logger:     fuseLogger,
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
		instance.logger.Fatalf("Failed to serve filesystem: %v", err)
	}

	wg.Wait()

    instance.logger.Info("Filesystem shutdown")
}

func (instance *Fuse) Close() error {
        instance.logger.Info("Shutting down filesystem")

		err := fuse.Unmount(instance.mountpoint)
		if err != nil {
			instance.logger.Fatalf("Failed to unmount filesystem: %v", err)
		}

        instance.logger.Info("Unmounted filesystem")

		err = instance.connection.Close()
		if err != nil {
			instance.logger.Fatalf("Failed to close connection: %v", err)
		}

        instance.logger.Info("Closed connection")

        return nil
}

func getNodeID(ID uint64) fuse.NodeID {
	return fuse.NodeID(ID)
}
