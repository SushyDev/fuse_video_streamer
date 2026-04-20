package grpc

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/client/provider/grpc/internal/filesystem"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	api "sushydev.github.io/stream_mount_api/go"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
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

	// Eagerly initiate the connection so that transient failures surface at
	// startup rather than on the first filesystem operation. gRPC connections
	// are lazy by default (IDLE); Connect() moves the state machine forward.
	connection.Connect()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Wait until the state changes from IDLE (i.e. a connection attempt has begun).
	connection.WaitForStateChange(ctx, connectivity.Idle)

	state := connection.GetState()
	if state == connectivity.TransientFailure || state == connectivity.Shutdown {
		logger.Warn(fmt.Sprintf("File system provider connection unhealthy (%s): %s", state, entry.Name))
	}

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
	return filepath.Join(provider.config.MountPoint, provider.name)
}

func (provider *provider) GetFileSystem() interfaces_filesystem_client.FileSystem {
	return provider.fileSystem
}

func (provider *provider) IsConnected() bool {
	if provider.connection == nil {
		return false
	}
	return provider.connection.GetState() == connectivity.Ready
}
