package cache

import (
	"io"
	"sync"
	"runtime/debug"
	
	"fuse_video_steamer/flags"
	"fuse_video_steamer/stream"

	lru "github.com/hashicorp/golang-lru/v2"
)

// CacheInterface defines the required methods for a cache implementation
type CacheInterface interface {
	io.ReaderAt
	io.Closer
}

// Cache implements a chunked caching layer over a stream
type Cache struct {
	lruCache  *lru.Cache[int64, []byte] // Store byte slices directly instead of custom struct
	stream    *stream.Stream
	fileSize  int64
	chunkSize int64
	mu        sync.Mutex // Simplified to a single mutex
}

// Ensure Cache implements CacheInterface
var _ CacheInterface = &Cache{}

// CacheOptions allows configuring the cache
type CacheOptions struct {
	ChunkSize int64
	CacheSize int
}

// DefaultCacheOptions returns sensible defaults
func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		ChunkSize: 4 * 1024 * 1024, // 4MB chunks
		CacheSize: 16,             // Cache 100 chunks
	}
}

// NewCache creates a new cache instance
func NewCache(stream *stream.Stream, fileSize int64, opts *CacheOptions) *Cache {
	if opts == nil {
		opts = DefaultCacheOptions()
	}
	
	// Create a new LRU cache with specified capacity
	lruCache, err := lru.New[int64, []byte](opts.CacheSize)
	if err != nil {
		// This error occurs only with invalid sizes
		panic(err)
	}

	return &Cache{
		lruCache:  lruCache,
		stream:    stream,
		fileSize:  fileSize,
		chunkSize: opts.ChunkSize,
	}
}

// getChunkData retrieves a chunk from cache or loads it if needed
func (c *Cache) getChunkData(chunkOffset int64) ([]byte, error) {
	// Check cache first without lock
	if data, found := c.lruCache.Get(chunkOffset); found {
		return data, nil
	}
	
	// Not in cache - lock and load
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check cache after acquiring lock
	if data, found := c.lruCache.Get(chunkOffset); found {
		return data, nil
	}
	
	// Calculate read size
	readSize := c.chunkSize
	if chunkOffset+readSize > c.fileSize {
		readSize = c.fileSize - chunkOffset
	}
	
	// Read data from stream
	buf := make([]byte, readSize)
	n, err := c.stream.ReadAt(buf, chunkOffset)
	if err != nil && err != io.EOF {
		return nil, err
	}
	
	// Trim buffer to actual read size
	buf = buf[:n]
	
	// Store in cache
	c.lruCache.Add(chunkOffset, buf)
	
	return buf, nil
}

// ReadAt implements io.ReaderAt
func (c *Cache) ReadAt(p []byte, off int64) (int, error) {
	if off >= c.fileSize {
		return 0, io.EOF
	}
	
	totalRead := 0
	currentOffset := off
	
	// Read chunks until we've filled the buffer or reached EOF
	for totalRead < len(p) && currentOffset < c.fileSize {
		// Calculate which chunk contains our data
		chunkOffset := (currentOffset / c.chunkSize) * c.chunkSize
		
		// Get chunk data
		chunkData, err := c.getChunkData(chunkOffset)
		if err != nil {
			return totalRead, err
		}
		
		// Calculate where in the chunk we should start
		chunkPos := int(currentOffset - chunkOffset)
		
		// Calculate how much we can copy from this chunk
		bytesToCopy := len(chunkData) - chunkPos
		if bytesToCopy > len(p) - totalRead {
			bytesToCopy = len(p) - totalRead
		}
		
		// Nothing left to read in this chunk
		if bytesToCopy <= 0 {
			break
		}
		
		// Copy data from chunk to output buffer
		copy(p[totalRead:totalRead+bytesToCopy], chunkData[chunkPos:chunkPos+bytesToCopy])
		
		// Update offsets
		totalRead += bytesToCopy
		currentOffset += int64(bytesToCopy)
	}
	
	// Return EOF if we didn't read the full amount requested
	var err error
	if totalRead < len(p) {
		err = io.EOF
	}
	
	return totalRead, err
}

// Close closes the underlying stream
func (cache *Cache) Close() error {
	if cache.stream != nil {
		err := cache.stream.Close()
		if  err != nil {
			return err
		}

		cache.stream = nil
	}

	if cache.lruCache != nil {
		cache.lruCache.Purge()
		cache.lruCache = nil
	}

	if *flags.GetIsDebug() {
		debug.FreeOSMemory()
	}

	return nil
}
