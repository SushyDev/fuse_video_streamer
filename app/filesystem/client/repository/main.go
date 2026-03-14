package repository

import (
	"fmt"

	interfaces_fuse "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/client/provider/grpc"
)

type clientRepository struct {
	loggerFactory interfaces_logger.LoggerFactory

	logger interfaces_logger.Logger

	clients []interfaces_fuse.Client
}

var _ interfaces_fuse.ClientRepository = &clientRepository{}

// NewWithClients constructs a clientRepository from a pre-built list of clients.
// Primarily intended for use in tests, where callers can supply mock clients directly
// without requiring a real config or gRPC targets.
func NewWithClients(clients []interfaces_fuse.Client) interfaces_fuse.ClientRepository {
	return &clientRepository{
		clients: clients,
	}
}

func New(config *config.Config, loggerFactory interfaces_logger.LoggerFactory, logger interfaces_logger.Logger) (interfaces_fuse.ClientRepository, error) {
	fileSystemProviders := config.GetFileServers()

	var providers []interfaces_fuse.Client
	for _, fileSystemProvider := range fileSystemProviders {
		provider, err := grpc.New(fileSystemProvider, config, loggerFactory)
		if err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return NewWithClients(providers), nil
}

func (repository *clientRepository) GetClientByName(name string) (interfaces_fuse.Client, error) {
	for _, client := range repository.clients {
		if client.GetName() == name {
			return client, nil
		}
	}

	return nil, fmt.Errorf("client with name %s not found", name)
}

func (repository *clientRepository) GetClients() ([]interfaces_fuse.Client, error) {
	return repository.clients, nil
}
