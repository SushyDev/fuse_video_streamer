package stream

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"debrid_drive/config"
	"debrid_drive/logger"
)

type PartialReader struct {
	url          string
	offset       int64
	Size         int64
	mu           sync.RWMutex
	cacheManager *cacheManager
	client       *http.Client
	closed       bool

	prefetchingChunks sync.Map // map[int64]struct{}

}

func NewPartialReader(url string, size int64) (*PartialReader, error) {
	cacheManager, err := newCacheManager(size)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	pr := &PartialReader{
		url: url,
		cacheManager: cacheManager,
		client:       &http.Client{},
		closed:       false,
	}

	if config.FetchFileSize {
		if err := pr.preFetchFileSize(); err != nil {
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

func (pr *PartialReader) Read(buffer []byte) (readBytes int, err error) {
	if pr.closed {
		return 0, fmt.Errorf("reader is closed")
	}

	offset := atomic.LoadInt64(&pr.offset)

	bufferSize, err := getBufferSize(offset, int64(len(buffer)), pr.Size)
	if err != nil {
		return 0, err
	}

	chunk := GetChunkByOffset(offset)

	data, err := chunk.getData(pr)
	if err != nil {
		return 0, err
	}

	readBytes, err = readData(chunk, data, buffer, bufferSize, offset)
	if err != nil {
		return 0, err
	}

	atomic.StoreInt64(&pr.offset, offset+int64(readBytes))

	chunkSize := chunk.getSize()
	chunkConsumed := float64(offset%chunkSize) / float64(chunkSize)
	if chunkConsumed > 0.75 {
		nextChunk := GetChunkByNumber(chunk.number + 1)
		nextChunkStart, _ := nextChunk.getRange()

		if nextChunkStart < pr.Size {
			go pr.preFetchChunkData(nextChunk)
		}
	}

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
		atomic.StoreInt64(&pr.offset, pr.Size)

		return pr.offset, io.EOF
	}

	atomic.StoreInt64(&pr.offset, newOffset)

	return pr.offset, nil
}

func (pr *PartialReader) Close() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.closed {
		return fmt.Errorf("reader is already closed")
	}

	if pr.cacheManager != nil {
		pr.cacheManager.Close()
	}

	if pr.client != nil {
		pr.client.CloseIdleConnections()
	}

	pr.closed = true

	return nil
}

func readData(chunk *cacheChunk, data []byte, buffer []byte, bufferSize int64, offset int64) (int, error) {
	chunkStart, chunkEnd := chunk.getRange()

	relativeOffset := offset - chunkStart
	if relativeOffset < 0 {
		relativeOffset = 0
	}

	start := relativeOffset
	end := relativeOffset + bufferSize

	if end > chunkEnd {
		end = chunkEnd
	}

	if start >= int64(len(data)) {
		return 0, io.EOF
	}

	if end > int64(len(data)) {
		end = int64(len(data))
	}

	requestedBytes := data[start:end]
	copySize := copy(buffer, requestedBytes)

	return copySize, nil
}

func getBufferSize(start int64, requestedSize int64, fileSize int64) (int64, error) {
	if start >= fileSize {
		return 0, io.EOF
	}

	if start+requestedSize > fileSize {
		requestedSize = fileSize - start
	}

	return requestedSize, nil
}
