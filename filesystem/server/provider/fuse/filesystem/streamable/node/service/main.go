package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fuse_video_steamer/filesystem/server/provider/fuse/filesystem/streamable/node"
	"fuse_video_steamer/filesystem/server/provider/fuse/interfaces"
	"fuse_video_steamer/filesystem/server/provider/fuse/registry"
	"fuse_video_steamer/logger"

	api "github.com/sushydev/stream_mount_api"
)

type Service struct {
	client   api.FileSystemServiceClient
	logger   *logger.Logger
	registry *registry.Registry

	mu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.StreamableNodeService = &Service{}

var clients = []api.FileSystemServiceClient{}

func New(client api.FileSystemServiceClient, logger *logger.Logger) (interfaces.StreamableNodeService, error) {
	ctx, cancel := context.WithCancel(context.Background())

	registry := registry.GetInstance(client)

	return &Service{
		client:   client,
		logger:   logger,
		registry: registry,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (service *Service) New(identifier uint64) (interfaces.StreamableNode, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	if service.isClosed() {
		return nil, fmt.Errorf("Service is closed")
	}

	logger, err := logger.NewLogger("Root Node")
	if err != nil {
		panic(err)
	}

	clientContext, cancel := context.WithTimeout(service.ctx, 30*time.Second)
	defer cancel()

	sizeResponse, err := service.client.GetFileInfo(clientContext, &api.GetFileInfoRequest{
		NodeId: identifier,
	})

	if err != nil {
		message := fmt.Sprintf("Failed to get video size for %d", identifier)
		service.logger.Error(message, err)
		return nil, err
	}

	newNode := node.New(service.client, logger, identifier, sizeResponse.Size)

	service.registry.Add(newNode)

	return newNode, nil
}

func (service *Service) Close() error {
	return nil
}

func (service *Service) isClosed() bool {
	select {
	case <-service.ctx.Done():
		return true
	default:
		return false
	}
}
