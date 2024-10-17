package stream

import (
	"debrid_drive/config"
	"debrid_drive/logger"
	"fmt"
)

func (pr *PartialReader) preFetchFileSize() error {
	resp, err := pr.client.Head(pr.url)
	if err != nil {
		return fmt.Errorf("HTTP HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.ContentLength < 0 {
		return fmt.Errorf("failed to get file size")
	}

	pr.Size = resp.ContentLength

	fmt.Printf("Fetched file size: %d\n", pr.Size)
	return nil
}

func (pr *PartialReader) preFetchHeaders() error {
	headSize := config.CacheChunkSize
	if headSize > pr.Size {
		headSize = pr.Size
	}

	start := 0
	end := headSize - 1

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)

	body, err := pr.fetchBytesInRange(rangeHeader)
	if err != nil {
		return fmt.Errorf("failed to prefetch headers: %w", err)
	}

	chunk := GetChunkByOffset(int64(start))

	pr.cacheManager.storeChunkDataInCache(chunk.number, body)

	fmt.Printf("Prefetched video headers in chunk %d\n", chunk.number)
	return nil
}

func (pr *PartialReader) preFetchTail() error {
	tailSize := config.CacheChunkSize
	if tailSize > pr.Size {
		tailSize = pr.Size
	}

	start := pr.Size - tailSize
	end := pr.Size - 1

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)

	body, err := pr.fetchBytesInRange(rangeHeader)
	if err != nil {
		return err
	}

	chunk := GetChunkByOffset(int64(start))

	pr.cacheManager.storeChunkDataInCache(chunk.number, body)

	fmt.Printf("Prefetched video tail in chunk %d\n", chunk.number)

	return nil
}

func (pr *PartialReader) preFetchChunkData(chunk *cacheChunk) {
	_, ok := pr.cacheManager.getChunkDataFromCache(chunk.number)
	if ok {
		return
	}

	prefetchChannel := make(chan struct{})
	_, loaded := pr.prefetchingChunks.LoadOrStore(chunk.number, prefetchChannel)
	if loaded {
		return
	}

	if pr.closed {
		pr.prefetchingChunks.Delete(chunk.number)
		close(prefetchChannel)
		return
	}

	go func(chunk *cacheChunk) {
		defer func() {
			pr.prefetchingChunks.Delete(chunk.number)
			close(prefetchChannel)
		}()

		data, err := pr.fetchAndCacheChunkData(chunk)
		if err != nil {
			logger.Logger.Errorf("Pre-fetching failed for chunk %d: %v", chunk.number, err)
			return
		}

		pr.cacheManager.storeChunkDataInCache(chunk.number, data)
	}(chunk)
}

func (pr *PartialReader) getChunkDataFromOngoingPrefetch(chunkNumber int64) ([]byte, bool) {
	prefetchChannel, exists := pr.prefetchingChunks.Load(chunkNumber)
	if !exists {
		return nil, false
	}

	<-prefetchChannel.(chan struct{})

	return pr.cacheManager.getChunkDataFromCache(chunkNumber)
}
