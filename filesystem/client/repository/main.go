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

func New(loggerFactory interfaces_logger.LoggerFactory, logger interfaces_logger.Logger) (interfaces_fuse.ClientRepository, error) {
	fileSystemProviders := config.GetFileServers()

	var providers []interfaces_fuse.Client
	for _, fileSystemProvider := range fileSystemProviders {
		provider, err := grpc.New(fileSystemProvider, loggerFactory)
		if err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return &clientRepository{
		loggerFactory: loggerFactory,

		logger: logger,

		clients: providers,
	}, nil
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
