package stream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"debrid_drive/config"
)

func (pr *PartialReader) fetchAndCacheChunkData(chunk *cacheChunk) ([]byte, error) {
	data, err := pr.fetchChunkData(chunk)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chunk %d: %w", chunk.number, err)
	}

	pr.cacheManager.storeChunkDataInCache(chunk.number, data)

	return data, nil
}

func (pr *PartialReader) fetchChunkData(chunk *cacheChunk) ([]byte, error) {
	start, end := chunk.getRange()

	if end > pr.Size {
		end = pr.Size
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end-1)

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
