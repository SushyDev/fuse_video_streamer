package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"debrid_drive/config"
)

func (pr *PartialReader) fetchFileSize() error {
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

	chunk := pr.storeAsChunkInCache(int64(start), body)

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

	chunk := pr.storeAsChunkInCache(start, body)

	fmt.Printf("Prefetched video tail in chunk %d\n", chunk.number)

	return nil
}

func (pr *PartialReader) fetchAndCacheChunk(buffer []byte, bufferSize int64, chunk cacheChunk) (int, error) {
	body, err := pr.fetchChunk(chunk)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch chunk %d: %w", chunk.number, err)
	}

	// Check if body is expected size (config.CachedChunkSize)
	if int64(len(body)) != config.CacheChunkSize {
		fmt.Printf("Chunk %d is not the expected size: %d/%d\n", chunk.number, len(body), config.CacheChunkSize)
	}

	pr.storeAsChunkInCache(chunk.startOffset, body)

	readBytes, err := pr.readFromCache(buffer, bufferSize, chunk, body)
	if err != nil {
		return readBytes, err
	}

	return readBytes, nil
}

func (pr *PartialReader) fetchChunk(chunk cacheChunk) ([]byte, error) {
	rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.startOffset, chunk.endOffset-1)

	return pr.fetchBytesInRange(rangeHeader)
}

func (pr *PartialReader) fetchBytesInRange(rangeHeader string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.FetchTimeout*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", pr.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Range", rangeHeader)

	resp, err := pr.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful range response.
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return body, nil
}
