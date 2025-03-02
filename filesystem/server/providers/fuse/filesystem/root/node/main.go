package node

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	directory_node_service_factory "fuse_video_steamer/filesystem/server/providers/fuse/filesystem/directory/node/service/factory"

	"fuse_video_steamer/api_clients"
	"fuse_video_steamer/filesystem/server/providers/fuse/interfaces"
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
	clients []vfs_api.FileSystemServiceClient

	mu     sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.RootNode = &Node{}

func New(directoryNodeService interfaces.DirectoryNodeService, logger *logger.Logger) (*Node, error) {
	ctx, cancel := context.WithCancel(context.Background())

	clients := api_clients.GetClients()

	return &Node{
		directoryNodeServiceFactory: directory_node_service_factory.New(),
		directoryNodeService: directoryNodeService,

		logger: logger,
		clients: clients,

		ctx: ctx,
		cancel: cancel,
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

	if fuseRoot.isClosed() {
		return nil, nil
	}

	return fuseRoot, nil
}

func (node *Node) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()
	
	if node.isClosed() {
		return nil, nil
	}

	client := node.clients[lookupRequest.Node-1]

	clientContext, cancel := context.WithTimeout(node.ctx, 30 * time.Second)
	defer cancel()

	response, err := client.Root(clientContext, &vfs_api.RootRequest{})
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

func (node *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	node.mu.RLock()
	defer node.mu.RUnlock()

	if node.isClosed() {
		return nil, nil
	}

	clientContext, cancel := context.WithTimeout(node.ctx, 30 * time.Second)
	defer cancel()

	var entries []fuse.Dirent
	for index, client := range node.clients {

		response, err := client.Root(clientContext, &vfs_api.RootRequest{})
		if err != nil {
			message := fmt.Sprintf("Failed to get root for client %d", index)
			node.logger.Error(message, err)
			return nil, err
		}

		entries = append(entries, fuse.Dirent{
			Name: response.Root.Name,
			Type: fuse.DT_Dir,
		})
	}

	return entries, nil
}

func (node *Node) Close() error {
	// node.mu.Lock()
	// defer node.mu.Unlock()

	node.cancel()

	node.directoryNodeService.Close()
	node.directoryNodeService = nil

	return nil
}

func (node *Node) isClosed() bool {
	select {
	case <-node.ctx.Done():
		return true
	default:
		return false
	}
}
