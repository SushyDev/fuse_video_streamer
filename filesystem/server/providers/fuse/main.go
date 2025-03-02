package fuse

import (
	"fmt"
	"strings"
	"time"

	filesystem_interfaces "fuse_video_steamer/filesystem/interfaces"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Server struct {
	mountpoint string
	connection *fuse.Conn
	fileSystem interfaces.FuseFileSystem

	logger     *logger.Logger
}

var _ filesystem_interfaces.FileSystemServer = &Server{}

func New(mountpoint string, connection *fuse.Conn, fileSystem interfaces.FuseFileSystem, logger *logger.Logger) *Server {
	return &Server{
		mountpoint: mountpoint,
		connection: connection,
		fileSystem: fileSystem,
		logger:     logger,
	}
}

func (server *Server) Serve() {
	fileSystemServer := fs.New(server.connection, nil)

	server.logger.Info("Serving filesystem")

	err := fileSystemServer.Serve(server.fileSystem)
	if err != nil {
		server.logger.Fatal("Failed to serve filesystem", err)
	}

	server.logger.Info("Filesystem shutdown")
}

func (instance *Server) Close() error {
	instance.fileSystem.Close()
	instance.fileSystem = nil

	err := instance.unmount()
	if err != nil {
		instance.logger.Error("Failed to unmount filesystem", err)
	}

	if instance.connection != nil {
		err := instance.connection.Close()
		if err != nil {
			instance.logger.Error("Failed to close connection", err)
		}

		instance.connection = nil
	}

	instance.logger.Info("Fuse closed")

	return nil
}

func (instance *Server) unmount() error {
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
