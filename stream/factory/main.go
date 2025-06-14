package factory

import (
	"context"
	"fmt"
	"time"

	filesystem_client_interfaces "fuse_video_steamer/filesystem/client/interfaces"
	"fuse_video_steamer/stream"
)

type CacheItem struct {
	url string
	expiration time.Time
}

type Factory struct {
	client filesystem_client_interfaces.Client

	cachedItem CacheItem

	context context.Context
	cancel  context.CancelFunc
}

func New(client filesystem_client_interfaces.Client) *Factory {
	ctx, cancel := context.WithCancel(context.Background())

	return &Factory{
		client:  client,

		context: ctx,
		cancel:  cancel,
	}
}

func (factory *Factory) NewStream(nodeIdentifier uint64, size uint64) (*stream.Stream, error) {
	if factory.isClosed() {
		return nil, fmt.Errorf("Factory is closed")
	}

	url, err := factory.getStreamUrl(nodeIdentifier)
	if err != nil {
		return nil, err
	}

	return stream.New(url, int64(size))
}

func (factory *Factory) getStreamUrl(identifier uint64) (string, error) {
	if factory.cachedItem.url != "" && factory.cachedItem.expiration.After(time.Now()) {
		return factory.cachedItem.url, nil
	}

	fileSystem := factory.client.GetFileSystem()

	url, err := fileSystem.GetStreamUrl(identifier)
	if err != nil {
		return "", fmt.Errorf("Failed to get video url for node with id %d", identifier)
	}

	factory.cachedItem = CacheItem{
		url: url,
		expiration: time.Now().Add(15 * time.Minute),
	}

	return url, nil
}

func (factory *Factory) Close() {
	factory.cancel()
}

func (factory *Factory) isClosed() bool {
	select {
	case <-factory.context.Done():
		return true
	default:
		return false
	}
}

// TODO - Wont be needed probably
