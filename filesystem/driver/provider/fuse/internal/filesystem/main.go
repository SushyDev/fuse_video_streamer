package filesystem

import (
	"sync/atomic"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	rootNodeService interfaces_fuse.RootNodeService

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_fuse.FuseFileSystem = &FileSystem{}

func New(rootNodeService interfaces_fuse.RootNodeService, logger interfaces_logger.Logger) interfaces_fuse.FuseFileSystem {
	metricsCollection := metrics.GetMetricsCollection()
	go metricsCollection.StartWebDebugger()

	return &FileSystem{
		rootNodeService: rootNodeService,

		logger: logger,
	}
}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	return fileSystem.rootNodeService.New()
}

func (fileSystem *FileSystem) Destroy() {
	fileSystem.logger.Info("Destroying filesystem")
	fileSystem.Close()
}

func (fileSystem *FileSystem) Close() error {
	if !fileSystem.closed.CompareAndSwap(false, true) {
		return nil
	}

	fileSystem.logger.Info("Closing")

	fileSystem.rootNodeService.Close()
	fileSystem.rootNodeService = nil

	registry.Close()

	fileSystem.logger.Info("Closed")

	return nil
}

func (fileSystem *FileSystem) IsClosed() bool {
	return fileSystem.closed.Load()
}
