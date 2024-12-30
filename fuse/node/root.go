package node

import (
	"context"
	"fmt"
	"os"
	"sync"

	"fuse_video_steamer/config"
	"fuse_video_steamer/logger"

	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

var _ fs.Handle = &Directory{}

var clients = []vfs_api.FileSystemServiceClient{}

type Root struct {
	logger     *logger.Logger
	mu         sync.RWMutex
}

func NewRoot() *Root {
	fuseLogger, _ := logger.NewLogger("Root Node")

	backoffConfig := backoff.DefaultConfig
	insecureCredentials := insecure.NewCredentials()
	connectParams := grpc.ConnectParams{
		Backoff: backoffConfig,
	}

	fileServers := config.GetFileServers()
	for _, fileServer := range fileServers {
		connection, err := grpc.NewClient(
			fileServer,
			grpc.WithTransportCredentials(insecureCredentials),
			grpc.WithConnectParams(connectParams),

		)

		if err != nil {
			fuseLogger.Error(fmt.Sprintf("Failed to connect to %s", fileServer), err)
			continue
		}

		client := vfs_api.NewFileSystemServiceClient(connection)

		fuseLogger.Info(fmt.Sprintf("Connected to %s", fileServer))

		clients = append(clients, client)
	}

	return &Root{
		logger: fuseLogger,
	}
}

var _ fs.Node = &Root{}

func (fuseRoot *Root) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	attr.Mode = os.ModeDir

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

var _ fs.NodeOpener = &Directory{}

func (fuseRoot *Root) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	return fuseRoot, nil
}

var _ fs.NodeRequestLookuper = &Root{}

func (fuseRoot *Root) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	client := clients[lookupRequest.Node-1]

	response, err := client.Root(ctx, &vfs_api.RootRequest{})
	if err != nil {
		fuseRoot.logger.Error("Failed to get root", err)
		return nil, err
	}

	return NewDirectory(client, response.Root.Identifier), nil
}

var _ fs.HandleReadDirAller = &Root{}

func (fuseRoot *Root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	var entries []fuse.Dirent
	for _, client := range clients {
		response, err := client.Root(ctx, &vfs_api.RootRequest{})
		if err != nil {
			fuseRoot.logger.Error("Failed to get root", err)
			return nil, err
		}

		entries = append(entries, fuse.Dirent{
			Name: response.Root.Name,
			Type: fuse.DT_Dir,
		})
	}

	return entries, nil
}
