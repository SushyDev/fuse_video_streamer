package filesystem

import (
	"sync/atomic"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	rootNodeService interfaces.RootNodeService

	logger  *logger.Logger

	closed atomic.Bool
}

var _ interfaces.FuseFileSystem = &FileSystem{}

func New(rootNodeService interfaces.RootNodeService) interfaces.FuseFileSystem {
	logger, err := logger.NewLogger("Filesystem")
	if err != nil {
		panic(err)
	}

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
