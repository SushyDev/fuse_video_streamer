package root

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	"fuse_video_streamer/healthcheck"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type node struct {
	fileSystemProviderRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_node.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory

	loggerFactory        interfaces_logger.LoggerFactory
	directoryNodeService interfaces_node.DirectoryNodeService

	logger interfaces_logger.Logger

	tree interfaces_node.Tree

	healthCheckNode *healthcheck.HealthCheckFileNode

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_node.RootNode = &node{}

func NewNode(
	fileSystemProviderRepository interfaces_filesystem_client.ClientRepository,
	directoryNodeServiceFactory interfaces_node.DirectoryNodeServiceFactory,
	directoryHandleServiceFactory interfaces_handle.DirectoryHandleServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	directoryNodeService interfaces_node.DirectoryNodeService,
	logger interfaces_logger.Logger,
	tree interfaces_node.Tree,
) (*node, error) {
	return &node{
		fileSystemProviderRepository: fileSystemProviderRepository,

		directoryNodeServiceFactory:   directoryNodeServiceFactory,
		directoryHandleServiceFactory: directoryHandleServiceFactory,

		loggerFactory:        loggerFactory,
		directoryNodeService: directoryNodeService,

		tree: tree,

		healthCheckNode: healthcheck.NewHealthCheckFileNode(),

		logger: logger,
	}, nil
}

func (node *node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	// Root directory with 0755 permissions
	attr.Mode = os.ModeDir | 0755
	attr.Inode = 1
	attr.Valid = 30 * time.Second

	return nil
}

func (node *node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	return node, nil
}

func (node *node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	// Check if looking up the health check file
	if lookupRequest.Name == healthcheck.HealthCheckFileName {
		return node.healthCheckNode, nil
	}

	client, err := node.fileSystemProviderRepository.GetClientByName(lookupRequest.Name)
	if err != nil {
		return nil, err
	}

	fileSystem := client.GetFileSystem()

	root, err := fileSystem.Root(client.GetName())
	if err != nil {
		message := fmt.Sprintf("failed to get root for client %s", lookupRequest.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	directoryNodeService, err := node.directoryNodeServiceFactory.NewService(client, node.tree)
	if err != nil {
		message := fmt.Sprintf("failed to create directory node service for client %s", lookupRequest.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	return directoryNodeService.NewNode(nil, root.GetId(), root.GetMode())
}

func (node *node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsClosed() {
		return nil, nil
	}

	clients, err := node.fileSystemProviderRepository.GetClients()
	if err != nil {
		return nil, err
	}

	var entries []fuse.Dirent

	// Add health check file
	entries = append(entries, fuse.Dirent{
		Name: healthcheck.HealthCheckFileName,
		Type: fuse.DT_File,
	})

	for _, client := range clients {
		fileSystem := client.GetFileSystem()
		_, err := fileSystem.Root(client.GetName())
		if err != nil {
			// Skip disconnected clients
			continue
		}

		entries = append(entries, fuse.Dirent{
			Name: client.GetName(),
			Type: fuse.DT_Dir,
		})
	}

	return entries, nil
}

func (node *node) Close() error {
	if !node.closed.CompareAndSwap(false, true) {
		return nil
	}

	node.mu.Lock()
	defer node.mu.Unlock()

	if node.healthCheckNode != nil {
		node.healthCheckNode.Close()
		node.healthCheckNode = nil
	}

	node.directoryNodeService.Close()
	node.directoryNodeService = nil

	return nil
}

func (node *node) IsClosed() bool {
	return node.closed.Load()
}
