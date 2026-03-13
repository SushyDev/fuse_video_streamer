package factory

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"fuse_video_streamer/config"
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/stream/drivers/http_ring_buffer"
)

type CacheItem struct {
	url        string
	expiration time.Time
}

type Factory struct {
	config *config.Config
	client interfaces_filesystem_client.Client

	loggerFactory interfaces_logger.LoggerFactory

	cachedItem CacheItem

	closed atomic.Bool
}

func New(
	config *config.Config,
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
) *Factory {
	return &Factory{
		config:        config,
		client:        client,
		loggerFactory: loggerFactory,
	}
}

func (factory *Factory) NewStream(nodeIdentifier uint64, size uint64) (*http_ring_buffer.Stream, error) {
	if factory.isClosed() {
		return nil, fmt.Errorf("factory is closed")
	}

	url, err := factory.getStreamURL(nodeIdentifier, 0)
	if err != nil {
		return nil, err
	}

	return http_ring_buffer.New(factory.config, factory.loggerFactory, url, int64(size))
}

func (factory *Factory) getStreamURL(identifier uint64, tries int) (string, error) {
	const maxRetries = 30
	const maxBackoff = 30 * time.Second

	if factory.cachedItem.url != "" && factory.cachedItem.expiration.After(time.Now()) {
		return factory.cachedItem.url, nil
	}

	if tries >= maxRetries {
		return "", fmt.Errorf("failed to get video url for node with id %d after %d retries", identifier, maxRetries)
	}

	fileSystem := factory.client.GetFileSystem()

	url, err := fileSystem.GetStreamUrl(identifier)

	backoffDuration := min(
		time.Duration(100*math.Pow(2, float64(tries)))*time.Millisecond,
		maxBackoff,
	)

	if err != nil {
		time.Sleep(backoffDuration)
		return factory.getStreamURL(identifier, tries+1)
	}

	if url == "" {
		time.Sleep(backoffDuration)
		return factory.getStreamURL(identifier, tries+1)
	}

	factory.cachedItem = CacheItem{
		url:        url,
		expiration: time.Now().Add(15 * time.Minute),
	}

	return url, nil
}

func (factory *Factory) Close() error {
	if !factory.closed.CompareAndSwap(false, true) {
		return nil
	}

	return nil
}

func (factory *Factory) isClosed() bool {
	return factory.closed.Load()
}
