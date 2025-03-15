package api_clients

import (
	"fmt"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/config"

	api "github.com/sushydev/stream_mount_api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/credentials/insecure"
)

func GetClients() (clients []api.FileSystemServiceClient) {
	logger, err := logger.NewLogger("API Clients")
	if err != nil {
		panic(err)
	}

	connectParams := grpc.ConnectParams{
		Backoff: backoff.DefaultConfig,
	}

	keepAliveParams := keepalive.ClientParameters{
		Time:                10,
		Timeout:             10,
		PermitWithoutStream: true,
	}

	insecureCredentials := insecure.NewCredentials()

	fileServers := config.GetFileServers()
	for _, fileServer := range fileServers {
		connection, err := grpc.NewClient(
			fileServer,
			grpc.WithConnectParams(connectParams),
			grpc.WithKeepaliveParams(keepAliveParams),
			grpc.WithTransportCredentials(insecureCredentials),
		)

		if err != nil {
			message := fmt.Sprintf("Failed to connect to %s", fileServer)
			logger.Error(message, err)
			continue
		}

		client := api.NewFileSystemServiceClient(connection)

		logger.Info(fmt.Sprintf("Connected to %s", fileServer))

		clients = append(clients, client)
	}

	return clients
}
