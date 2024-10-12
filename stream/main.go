package stream

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	lru "github.com/hashicorp/golang-lru"

	"debrid_drive/config"
	"debrid_drive/logger"
)

type PartialReader struct {
	url      string
	offset   int64
	offsetMu sync.Mutex
	Size     int64
	Chunks   int64
	mu       sync.RWMutex
	cache    *lru.Cache
	cacheMu  sync.Mutex
	client   *http.Client
	closed   bool
}

func NewPartialReader(url string, size int64) (*PartialReader, error) {
	pr := &PartialReader{
		url:    url,
		client: &http.Client{},
	}

	cache, err := lru.New(config.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	pr.cache = cache

	if config.FetchFileSize {
		if err := pr.fetchFileSize(); err != nil {
			logger.Logger.Errorf("Failed to fetch file size: %v", err)
		}
	} else {
		pr.Size = size
	}

	if config.FetchHeaders {
		if err := pr.preFetchHeaders(); err != nil {
			logger.Logger.Errorf("Failed to prefetch headers: %v", err)
		}
	}

	if config.FetchTail {
		if err := pr.preFetchTail(); err != nil {
			logger.Logger.Errorf("Failed to prefetch tail: %v", err)
		}
	}

	return pr, nil
}

func (pr *PartialReader) setOffset(offset int64) {
	pr.offsetMu.Lock()
	defer pr.offsetMu.Unlock()

	pr.offset = offset
}

func (pr *PartialReader) getOffset() int64 {
	pr.offsetMu.Lock()
	defer pr.offsetMu.Unlock()

	return pr.offset
}

func (pr *PartialReader) Read(buffer []byte) (n int, err error) {
	currentOffset := pr.getOffset()

	if pr.closed {
		return 0, fmt.Errorf("reader is closed")
	}

	start, bufferSize, err := pr.calculateReadBoundaries(currentOffset, int64(len(buffer)))
	if err != nil {
		return 0, err
	}

	chunk := pr.getChunkByOffset(start)

	var readBytes int
	var readErr error

	cachedData, ok := pr.getFromCache(chunk.number)
	if ok {
		readBytes, readErr = pr.readFromCache(buffer, bufferSize, chunk, cachedData)
		if readErr != nil {
			logger.Logger.Errorf("readFromCache failed: %v", readErr)
			return readBytes, readErr
		}
	} else {
		readBytes, readErr = pr.fetchAndCacheChunk(buffer, bufferSize, chunk)

		if readErr != nil {
			logger.Logger.Errorf("fetchAndCacheChunk failed: %v", readErr)
			return readBytes, readErr
		}
	}

	pr.setOffset(pr.offset)

	return readBytes, nil
}

func (pr *PartialReader) Seek(offset int64, whence int) (int64, error) {
	if pr.closed {
		return 0, fmt.Errorf("reader is closed")
	}

	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = pr.offset + offset
	case io.SeekEnd:
		newOffset = pr.Size + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("negative position is invalid")
	}

	if newOffset >= pr.Size {
		pr.setOffset(pr.Size)

		return pr.offset, io.EOF
	}

	pr.setOffset(newOffset)

	return pr.offset, nil
}

func (pr *PartialReader) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.closed {
		return fmt.Errorf("reader is already closed")
	}

	pr.cache.Purge()
	pr.closed = true

	if pr.client != nil {
		pr.client.CloseIdleConnections()
	}

	logger.Logger.Infof("Closed PartialReader for URL: %s", pr.url)

	return nil
}

func (pr *PartialReader) calculateReadBoundaries(start, requestedSize int64) (int64, int64, error) {
	if start >= pr.Size {
		return 0, 0, io.EOF
	}

	if start+requestedSize > pr.Size {
		requestedSize = pr.Size - start
	}

	return start, requestedSize, nil
}
