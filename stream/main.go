package stream

import (
	"debrid_drive/config"
	"fmt"
	"io"
	"net/http"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

type PartialReader struct {
	url    string
	offset int64
	Size   int64
	mu     sync.Mutex
	cache  *lru.Cache
	client *http.Client
}

func NewPartialReader(url string) (*PartialReader, error) {
	pr := &PartialReader{
		url:    url,
		client: &http.Client{},
	}

	cache, err := lru.New(config.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	pr.cache = cache

	if err := pr.fetchFileSize(); err != nil {
		return nil, fmt.Errorf("failed to fetch file size: %w", err)
	}

	if err := pr.preFetchHeaders(); err != nil {
		return nil, fmt.Errorf("failed to prefetch headers: %w", err)
	}

	if err := pr.preFetchTail(); err != nil {
		return nil, fmt.Errorf("failed to prefetch tail: %w", err)
	}

	return pr, nil
}

func (pr *PartialReader) Read(buffer []byte) (n int, err error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	// Check if the current offset is beyond the file size
	if pr.offset >= pr.Size {
		return 0, io.EOF
	}

	bufferSize := int64(len(buffer))
	requestedReadSize := calculateReadSize(pr.offset, bufferSize, pr.Size)
	chunk := getChunkByStartOffset(pr.offset, pr.Size)

	var readBytes int
	var readErr error

	cachedData, ok := pr.cache.Get(chunk.number)
	if ok {
		readBytes, readErr = pr.readFromCache(buffer, requestedReadSize, chunk, cachedData.([]byte))
	} else {
		readBytes, readErr = pr.fetchAndCacheChunk(buffer, requestedReadSize, chunk)
	}

	if readErr != nil {
		return readBytes, readErr
	}

	pr.offset += int64(readBytes)
	return readBytes, nil
}

func (pr *PartialReader) Seek(offset int64, whence int) (int64, error) {
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
		pr.offset = pr.Size
		return pr.offset, io.EOF
	}

	pr.offset = newOffset
	return pr.offset, nil
}

func (pr *PartialReader) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.cache.Purge()

	return nil
}

func calculateReadSize(offset int64, bufferSize int64, fileSize int64) int64 {
	if offset+bufferSize > fileSize {
		return fileSize - offset
	}

	return bufferSize
}
