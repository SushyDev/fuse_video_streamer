package stream

import (
	"fmt"
	"io"
	"net/http"

	"debrid_drive/config"
)

// fetchFileSize retrieves the total size of the file using an HTTP HEAD request.
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

// PreFetchHeaders fetches the initial portion of the file (typically metadata) to help with quick video startup.
func (pr *PartialReader) PreFetchHeaders() error {
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

    chunk := storeAsChunkInCache(pr, int64(start), body)

	fmt.Printf("Prefetched video headers in chunk %d\n", chunk.number)
	return nil
}

// PreFetchTail fetches the last few KBs of the file (tail end) to help with fast seeking in MKV files.
func (pr *PartialReader) PreFetchTail() error {
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

    chunk := storeAsChunkInCache(pr, start, body)

	fmt.Printf("Prefetched video tail in chunk %d\n", chunk.number)

	return nil
}

// fetchAndCacheChunk fetches a chunk from the server, caches it, and reads the requested bytes.
func (pr *PartialReader) fetchAndCacheChunk(buffer []byte, requestedReadSize int64, chunk Chunk) (int, error) {
    rangeHeader := fmt.Sprintf("bytes=%d-%d", chunk.startOffset, chunk.endOffset - 1)

	body, err := pr.fetchBytesInRange(rangeHeader)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch chunk %d: %w", chunk.number, err)
	}

	// Check if body is expected size (config.CachedCHunkSize)
	if int64(len(body)) != config.CacheChunkSize {
		fmt.Printf("Chunk %d is not the expected size: %d/%d\n", chunk.number, len(body), config.CacheChunkSize)
	}

    storeAsChunkInCache(pr, chunk.startOffset, body)

	// Read the requested bytes from the fetched chunk.
	readBytes, err := pr.readFromCache(buffer, requestedReadSize, chunk, body)
	if err != nil {
		return readBytes, err
	}

	return readBytes, nil
}

// fetchBytesInRange performs an HTTP GET request with the specified Range header.
func (pr *PartialReader) fetchBytesInRange(rangeHeader string) ([]byte, error) {
	req, err := http.NewRequest("GET", pr.url, nil)
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
