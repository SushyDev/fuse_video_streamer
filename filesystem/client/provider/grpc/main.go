package grpc

import (
	"fuse_video_steamer/config"
	"fuse_video_steamer/logger"
	"fuse_video_steamer/filesystem/client/interfaces"
	"fuse_video_steamer/filesystem/client/provider/grpc/internal/filesystem"

	api "github.com/sushydev/stream_mount_api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type provider struct{
	name string
	target string
	fileSystem interfaces.FileSystem
}

var _ interfaces.Client = &provider{}

func New(entry config.FileSystemProvider) (interfaces.Client, error) {
	connectParams := grpc.ConnectParams{
		Backoff: backoff.DefaultConfig,
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                10,
		Timeout:             10,
		PermitWithoutStream: true,
	}

	insecureCredentials := insecure.NewCredentials()

	connection, err := grpc.NewClient(
		entry.Target,
		grpc.WithConnectParams(connectParams),
		grpc.WithKeepaliveParams(keepAliveParams),
		grpc.WithTransportCredentials(insecureCredentials),
	)

	if err != nil {
		return nil, err
	}

	client := api.NewFileSystemServiceClient(connection)

	logger, err := logger.NewLogger("File System")
	if err != nil {
		return nil, err
	}

	fileSystem := filesystem.New(client, logger)

	return &provider{
		name: entry.Name,
		target: entry.Target,
		fileSystem: fileSystem,
	}, nil
}

func (p *provider) GetName() string {
	return p.name
}

func (p *provider) GetFileSystem() interfaces.FileSystem {
	return p.fileSystem
}
