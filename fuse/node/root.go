package node

import (
	"context"
	"fmt"
	"os"
	"sync"

	"fuse_video_steamer/fuse/interfaces"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

type Root struct {
	nodeService interfaces.NodeService
	logger *logger.Logger
	mu     sync.RWMutex
	clients []vfs_api.FileSystemServiceClient
}

var _ interfaces.Root = &Root{}

func NewRoot(service interfaces.NodeService, logger *logger.Logger, clients []vfs_api.FileSystemServiceClient) *Root {
	return &Root{
		nodeService: service,
		logger: logger,
		clients: clients,
	}
}

func (fuseRoot *Root) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	attr.Mode = os.ModeDir

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

func (fuseRoot *Root) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	return fuseRoot, nil
}

func (fuseRoot *Root) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	client := fuseRoot.clients[lookupRequest.Node-1]

	response, err := client.Root(ctx, &vfs_api.RootRequest{})
	if err != nil {
		message := fmt.Sprintf("Failed to lookup %s", lookupRequest.Name)
		fuseRoot.logger.Error(message, err)
		return nil, err
	}

	return fuseRoot.nodeService.NewDirectory(client, response.Root.Identifier)
}

func (fuseRoot *Root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
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
