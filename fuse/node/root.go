package node

import (
	"context"
	"log"
	"os"
	"sync"

	"fuse_video_steamer/logger"

	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var _ fs.Handle = &Directory{}

var hosts = []string{
	"localhost:6969",
}

var clients = []vfs_api.FileSystemServiceClient{}

type Root struct {
	identifier uint64
	logger     *zap.SugaredLogger
	mu         sync.RWMutex
}

func NewRoot() *Root {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

    for _, host := range hosts {
        connection, err := grpc.Dial(host, grpc.WithInsecure())
        if err != nil {
            log.Fatalf("Failed to connect to %s: %v", host, err)
        }

        client := vfs_api.NewFileSystemServiceClient(connection)

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

    client := clients[lookupRequest.Node - 1]

    response, err := client.Root(ctx, &vfs_api.RootRequest{})
    if err != nil {
        log.Fatalf("Failed to get root: %v", err)
        return nil, err
    }

    return NewDirectory(client, response.Root.Identifier), nil
}

var _ fs.HandleReadDirAller = &Root{}

type DirectoryResponse struct {
	Identifier uint64 `json:"identifier"`
	Name       string `json:"name"`
}

func (fuseRoot *Root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseRoot.mu.RLock()
	defer fuseRoot.mu.RUnlock()

	var entries []fuse.Dirent
	for _, client := range clients {
        response, err := client.Root(ctx, &vfs_api.RootRequest{})
        if err != nil {
            log.Fatalf("Failed to get root: %v", err)
            return nil, err
        }

		entries = append(entries, fuse.Dirent{
			Name:  response.Root.Name,
			Type:  fuse.DT_Dir,
		})
	}

	return entries, nil
}
