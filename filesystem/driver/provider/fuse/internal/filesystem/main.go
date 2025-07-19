package filesystem

import (
	"fmt"
	"sync/atomic"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"

	"github.com/anacrolix/fuse/fs"
)

type FileSystem struct {
	tree interfaces_fuse.Tree

	rootNode interfaces_fuse.RootNode

	logger interfaces_logger.Logger

	closed atomic.Bool
}

var _ interfaces_fuse.FuseFileSystem = &FileSystem{}

func New(tree interfaces_fuse.Tree, rootNodeService interfaces_fuse.RootNodeService, logger interfaces_logger.Logger) (interfaces_fuse.FuseFileSystem, error) {
	metricsCollection := metrics.GetMetricsCollection()
	go metricsCollection.StartWebDebugger()

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if rootNodeService == nil {
		return nil, fmt.Errorf("root Node Service cannot be nil")
	}

	rootNode, err := rootNodeService.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create root node: %w", err)
	}

	fileSystem := &FileSystem{
		tree: tree,

		rootNode: rootNode,

		logger: logger,
	}

	return fileSystem, nil
}

func (fileSystem *FileSystem) Root() (fs.Node, error) {
	return fileSystem.rootNode, nil
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

	fileSystem.rootNode.Close()
	registry.Close()

	fileSystem.logger.Info("Closed")

	return nil
}

func (fileSystem *FileSystem) IsClosed() bool {
	return fileSystem.closed.Load()
}
