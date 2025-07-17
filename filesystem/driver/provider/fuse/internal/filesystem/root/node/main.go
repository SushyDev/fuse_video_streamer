package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_fuse "fuse_video_streamer/filesystem/driver/provider/fuse/internal/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type node struct {
	fileSystemProviderRepository interfaces_filesystem_client.ClientRepository

	directoryNodeServiceFactory   interfaces_fuse.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory

	loggerFactory        interfaces_logger.LoggerFactory
	directoryNodeService interfaces_fuse.DirectoryNodeService

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_fuse.RootNode = &node{}

func New(
	fileSystemProviderRepository interfaces_filesystem_client.ClientRepository,
	directoryNodeServiceFactory interfaces_fuse.DirectoryNodeServiceFactory,
	directoryHandleServiceFactory interfaces_fuse.DirectoryHandleServiceFactory,
	loggerFactory interfaces_logger.LoggerFactory,
	directoryNodeService interfaces_fuse.DirectoryNodeService,
	logger interfaces_logger.Logger,
) (*node, error) {
	return &node{
		fileSystemProviderRepository: fileSystemProviderRepository,

		directoryNodeServiceFactory:   directoryNodeServiceFactory,
		directoryHandleServiceFactory: directoryHandleServiceFactory,

		loggerFactory:        loggerFactory,
		directoryNodeService: directoryNodeService,

		logger: logger,
	}, nil
}

func (node *node) GetIdentifier() uint64 {
	return 0
}

func (node *node) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	attr.Mode = os.ModeDir

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
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

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

	directoryNodeService, err := node.directoryNodeServiceFactory.New(client)
	if err != nil {
		return nil, err
	}

	return directoryNodeService.New(root.GetId())
}

func (node *node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	if node.IsClosed() {
		return nil, nil
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	clients, err := node.fileSystemProviderRepository.GetClients()
	if err != nil {
		return nil, err
	}

	var entries []fuse.Dirent
	for _, client := range clients {
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

	node.directoryNodeService.Close()
	node.directoryNodeService = nil

	return nil
}

func (node *node) IsClosed() bool {
	return node.closed.Load()
}
