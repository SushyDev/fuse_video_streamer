package repository

import (
	"fmt"

	"fuse_video_streamer/config"
	"fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/filesystem/client/provider/grpc"
)

type clientRepository struct {
	logger *logger.Logger
	clients []interfaces.Client
}

var _ interfaces.ClientRepository = &clientRepository{}

func New() (interfaces.ClientRepository, error) {
	logger, err := logger.NewLogger("Provider Repository")
	if err != nil {
		return nil, err
	}

	fileSystemProviders := config.GetFileServers()

	var providers []interfaces.Client
	for _, fileSystemProvider := range fileSystemProviders {
		provider, err := grpc.New(fileSystemProvider)
		if err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return &clientRepository{
		logger: logger,
		clients: providers,
	}, nil
}

func (repository *clientRepository) GetClientByName(name string) (interfaces.Client, error) {
	for _, client := range repository.clients {
		if client.GetName() == name {
			return client, nil
		}
	}

	return nil, fmt.Errorf("Client with name %s not found", name)
}

func (repository *clientRepository) GetClients() ([]interfaces.Client, error) {
	return repository.clients, nil
}
