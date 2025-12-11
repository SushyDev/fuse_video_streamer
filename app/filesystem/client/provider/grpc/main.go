package grpc

import (
	"fmt"
	"path/filepath"

	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/client/provider/grpc/internal/filesystem"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	api "sushydev.github.io/stream_mount_api/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type provider struct {
	config *config.Config

	name       string
	target     string
	connection *grpc.ClientConn
	fileSystem interfaces_filesystem_client.FileSystem
}

var _ interfaces_filesystem_client.Client = &provider{}

func New(entry config.FileSystemProvider, config *config.Config, loggerFactory interfaces_logger.LoggerFactory) (interfaces_filesystem_client.Client, error) {
	connectParams := grpc.ConnectParams{
		Backoff: backoff.DefaultConfig,
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                10,
		Timeout:             10,
		PermitWithoutStream: true,
	}

	insecureCredentials := insecure.NewCredentials()

	connection, err := grpc.NewClient(entry.Target,
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
		config:     config,
		name:       entry.Name,
		target:     entry.Target,
		connection: connection,
		fileSystem: fileSystem,
	}, nil
}

func (provider *provider) GetName() string {
	return provider.name
}

func (provider *provider) GetDirectory() string {
	return filepath.Join(provider.config.GetMountPoint(), provider.name)
}

func (provider *provider) GetFileSystem() interfaces_filesystem_client.FileSystem {
	return provider.fileSystem
}

func (provider *provider) IsConnected() bool {
	if provider.connection == nil {
		return false
	}
	state := provider.connection.GetState()
	// Check if connection is in READY state
	return state.String() == "READY"
}
