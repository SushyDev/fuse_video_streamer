package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"
	filesystem_provider_repository "fuse_video_streamer/filesystem/client/repository"
	directory_node_service_factory "fuse_video_streamer/filesystem/server/provider/fuse/filesystem/directory/node/service/factory"
	"fuse_video_streamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_streamer/logger"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type node struct {
	fileSystemProviderRepository filesystem_client_interfaces.ClientRepository

	directoryNodeServiceFactory   interfaces.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces.DirectoryHandleServiceFactory
	directoryNodeService          interfaces.DirectoryNodeService

	logger  *logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces.RootNode = &node{}

func New(directoryNodeService interfaces.DirectoryNodeService, logger *logger.Logger) (*node, error) {
	fileSystemProviderRepository, err := filesystem_provider_repository.New()
	if err != nil {
		return nil, err
	}

	return &node{
		fileSystemProviderRepository: fileSystemProviderRepository,

		directoryNodeServiceFactory: directory_node_service_factory.New(),
		directoryNodeService:        directoryNodeService,

		logger:  logger,
	}, nil
}

func (node *node) GetIdentifier() uint64 {
	return 0
}

func (node *node) Attr(ctx context.Context, attr *fuse.Attr) error {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsClosed() {
		return syscall.ENOENT
	}

	attr.Mode = os.ModeDir

	return nil
}

func (node *node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	return node, nil
}

func (node *node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	client, err := node.fileSystemProviderRepository.GetClientByName(lookupRequest.Name)
	if err != nil {
		return nil, err
	}

	fileSystem := client.GetFileSystem()

	root, err := fileSystem.Root(client.GetName())
	if err != nil {
		message := fmt.Sprintf("Failed to get root for client %s", lookupRequest.Name)
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
