package grpc

import (
	"fmt"

	"fuse_video_streamer/config"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/client/provider/grpc/internal/filesystem"

	api "github.com/sushydev/stream_mount_api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type provider struct {
	name       string
	target     string
	fileSystem interfaces_filesystem_client.FileSystem
}

var _ interfaces_filesystem_client.Client = &provider{}

func New(entry config.FileSystemProvider, loggerFactory interfaces_logger.LoggerFactory) (interfaces_filesystem_client.Client, error) {
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

	logger, err := loggerFactory.NewLogger("File System")
	if err != nil {
		return nil, err
	}

	fileSystem := filesystem.New(client, logger)

	// TODO healthcheck endpoint
	logger.Info(fmt.Sprintf("Connected to file system provider:	%s", entry.Name))

	return &provider{
		name:       entry.Name,
		target:     entry.Target,
		fileSystem: fileSystem,
	}, nil
}

func (p *provider) GetName() string {
	return p.name
}

func (p *provider) GetFileSystem() interfaces_filesystem_client.FileSystem {
	return p.fileSystem
}
