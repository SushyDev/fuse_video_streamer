package node

import (
	"context"
	"fmt"
	"os"
	"sync"

	directory_node_service_factory "fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/node/service/factory"

	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
	"fuse_video_steamer/api_clients"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Node struct {
	directoryNodeServiceFactory interfaces.DirectoryNodeServiceFactory
	directoryHandleServiceFactory interfaces.DirectoryHandleServiceFactory
	directoryNodeService         interfaces.DirectoryNodeService

	logger *logger.Logger
	mu     sync.RWMutex
	clients []vfs_api.FileSystemServiceClient
}

var _ interfaces.RootNode = &Node{}

func New(directoryNodeService interfaces.DirectoryNodeService, logger *logger.Logger) (*Node, error) {
	clients := api_clients.GetClients()

	return &Node{
		directoryNodeServiceFactory: directory_node_service_factory.New(),
		directoryNodeService: directoryNodeService,

		logger: logger,
		clients: clients,
	}, nil
}

func (fuseRoot *Node) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	attr.Mode = os.ModeDir

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (fuseRoot *Node) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	return fuseRoot, nil
}

func (node *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	client := node.clients[lookupRequest.Node-1]

	response, err := client.Root(ctx, &vfs_api.RootRequest{})
	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", lookupRequest.Name)
		node.logger.Error(message, err)
		return nil, err
	}

	directoryNodeService, err := node.directoryNodeServiceFactory.New(client)
	if err != nil {
		return nil, err
	}

	return directoryNodeService.New(response.Root.Identifier)
}

func (fuseRoot *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	var entries []fuse.Dirent
	for index, client := range fuseRoot.clients {
		response, err := client.Root(ctx, &vfs_api.RootRequest{})
		if err != nil {
			message := fmt.Sprintf("Failed to get root for client %d", index)
			fuseRoot.logger.Error(message, err)
			return nil, err
		}

		entries = append(entries, fuse.Dirent{
			Name: response.Root.Name,
			Type: fuse.DT_Dir,
		})
	}

	return entries, nil
}
