package factory

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"fuse_video_streamer/config"
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/stream/drivers/http_ring_buffer"
	interfaces_stream "fuse_video_streamer/stream/interfaces"
)

var _ interfaces_stream.StreamFactory = &Factory{}

type CacheItem struct {
	url        string
	expiration time.Time
}

type Factory struct {
	config *config.Config
	client interfaces_filesystem_client.Client

	loggerFactory interfaces_logger.LoggerFactory

	mu         sync.Mutex
	cachedItem CacheItem

	closed atomic.Bool

	// newTimer is called to produce the backoff timer channel. In production
	// this is time.After; tests may inject a zero-duration replacement so the
	// retry loop completes immediately.
	newTimer func(d time.Duration) <-chan time.Time
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
		newTimer:      time.After,
	}
}

func (factory *Factory) NewStream(ctx context.Context, nodeIdentifier uint64, size uint64) (interfaces_stream.Stream, error) {
	if factory.isClosed() {
		return nil, fmt.Errorf("factory is closed")
	}

	url, err := factory.getStreamURL(ctx, nodeIdentifier)
	if err != nil {
		return nil, err
	}

	return http_ring_buffer.New(factory.config, factory.loggerFactory, url, int64(size))
}

func (factory *Factory) getStreamURL(ctx context.Context, identifier uint64) (string, error) {
	const maxRetries = 30
	const maxBackoff = 30 * time.Second

	for tries := 0; tries < maxRetries; tries++ {
		factory.mu.Lock()
		cached := factory.cachedItem
		factory.mu.Unlock()

		if cached.url != "" && cached.expiration.After(time.Now()) {
			return cached.url, nil
		}

		fileSystem := factory.client.GetFileSystem()

		url, err := fileSystem.GetStreamUrl(identifier)
		if err != nil {
			// Permanent filesystem errors (e.g. ENOENT — node does not exist) must
			// not be retried; no amount of waiting will make the node appear.
			if _, isPermanent := err.(syscall.Errno); isPermanent {
				return "", err
			}
		} else if url != "" {
			factory.mu.Lock()
			factory.cachedItem = CacheItem{
				url:        url,
				expiration: time.Now().Add(15 * time.Minute),
			}
			factory.mu.Unlock()

			return url, nil
		}

		backoffDuration := min(
			time.Duration(100*math.Pow(2, float64(tries)))*time.Millisecond,
			maxBackoff,
		)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-factory.newTimer(backoffDuration):
		}
	}

	return "", syscall.EAGAIN
}

func (factory *Factory) Close() error {
	if !factory.closed.CompareAndSwap(false, true) {
		return nil
	}

	return nil
}

// SetNewTimer replaces the timer function used for backoff delays. Tests use
// this to inject an instant timer so retry loops complete immediately.
func (factory *Factory) SetNewTimer(fn func(d time.Duration) <-chan time.Time) {
	factory.newTimer = fn
}

func (factory *Factory) isClosed() bool {
	return factory.closed.Load()
}
