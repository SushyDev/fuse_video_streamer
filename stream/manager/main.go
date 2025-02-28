package stream_manager

import (
	"context"
	"fmt"
	"time"

	"fuse_video_steamer/stream"
	"fuse_video_steamer/vfs_api"
)

type Manager struct {
	nodeIdentifier uint64
	size           uint64

	client  vfs_api.FileSystemServiceClient
	streams stream.Map
}

func NewManager(client vfs_api.FileSystemServiceClient, nodeIdentifier uint64, size uint64) *Manager {
	return &Manager{
		nodeIdentifier: nodeIdentifier,
		size:           size,

		client: client,
		streams: stream.Map{},
	}
}

func (manager *Manager) NewStream(pid uint32) (*stream.Stream, error) {
	clientContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := manager.client.GetVideoUrl(clientContext, &vfs_api.GetVideoUrlRequest{
		Identifier: manager.nodeIdentifier,
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to get video url for node with id %d", manager.nodeIdentifier)
	}

	newStream := stream.NewStream(response.Url, int64(manager.size))

	manager.streams.Store(pid, newStream)

	return newStream, nil
}

func (manager *Manager) GetStream(pid uint32) (*stream.Stream, bool) {
	existingStream, ok := manager.streams.Load(pid)
	if ok {
		if !existingStream.IsClosed() {
			return existingStream, true
		}

		manager.streams.Delete(pid)
	}

	return nil, false
}

func (manager *Manager) GetOrCreateStream(pid uint32) (*stream.Stream, error) {
	stream, ok := manager.GetStream(pid)
	if ok {
		return stream, nil
	}

	newStream, err := manager.NewStream(pid)
	if err != nil {
		return nil, err
	}

	return newStream, nil
}

func (manager *Manager) DeleteStream(pid uint32) {
	stream, ok := manager.streams.Load(pid)
	if !ok {
		return
	}

	stream.Close()

	manager.streams.Delete(pid)
}
