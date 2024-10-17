package stream

import (
	"fmt"
	"sync"

	"github.com/hashicorp/golang-lru"

	"debrid_drive/config"
)

type cacheManager struct {
	cache  *lru.Cache
	mu     sync.RWMutex
	closed bool
}

func newCacheManager(fileSize int64) (*cacheManager, error) {
	cacheSize := fileSize / config.CacheChunkSize
	if cacheSize < 1 {
		cacheSize = 1
	}

	cache, err := lru.New(int(cacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	return &cacheManager{
		cache:  cache,
		closed: false,
	}, nil
}

func (cacheManager *cacheManager) storeChunkDataInCache(chunkNumber int64, data []byte) bool {
	if cacheManager.closed {
		return false
	}

	cacheManager.mu.Lock()
	cacheManager.cache.Add(chunkNumber, data)
	cacheManager.mu.Unlock()

	return true
}

func (cacheManager *cacheManager) getChunkDataFromCache(chunkNumber int64) ([]byte, bool) {
	if cacheManager.closed {
		return nil, false
	}

	cacheManager.mu.Lock()
	data, ok := cacheManager.cache.Get(chunkNumber)
	cacheManager.mu.Unlock()

	if !ok {
		return nil, false
	}

	return data.([]byte), true
}

func (cacheManager *cacheManager) Close() {
	cacheManager.mu.Lock()
	cacheManager.cache.Purge()
	cacheManager.mu.Unlock()

	cacheManager.closed = true
}
