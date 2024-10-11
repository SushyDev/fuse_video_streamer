package stream

import (
	"debrid_drive/config"
	"errors"
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
	cache  *lru.Cache
	// IsMkv  bool // Flag to indicate if the stream is MKV
	muRead  sync.Mutex
	muSeek  sync.Mutex
	muFetch sync.Mutex
	muCache sync.Mutex
	client  *http.Client
}

// NewPartialReader initializes a new PartialReader
func NewPartialReader(url string) (*PartialReader, error) {
	pr := &PartialReader{
		url:    url,
		client: &http.Client{},
	}

	// Validate configuration parameters
	if config.CacheSize <= 0 {
		return nil, errors.New("CacheSize must be positive")
	}

	if config.CacheChunkSize <= 0 {
		return nil, errors.New("CacheChunkSize must be positive")
	}

	cache, err := lru.New(config.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	pr.cache = cache

	// Fetch the total size of the file.
	if err := pr.fetchFileSize(); err != nil {
		return nil, fmt.Errorf("failed to fetch file size: %w", err)
	}

	// Pre-fetch headers for faster video startup
	if err := pr.PreFetchHeaders(); err != nil {
		return nil, fmt.Errorf("failed to prefetch headers: %w", err)
	}

	// Pre-fetch tail for faster seeking in MKV files
	if err := pr.PreFetchTail(); err != nil {
		return nil, fmt.Errorf("failed to prefetch tail: %w", err)
	}

	return pr, nil
}

func (pr *PartialReader) Read(buffer []byte) (n int, err error) {
	pr.muRead.Lock()
	defer pr.muRead.Unlock()

	// Check if the current offset is beyond the file size
	if pr.offset >= pr.Size {
		return 0, io.EOF
	}

	bufferSize := int64(len(buffer))
	requestedReadSize := calculateReadSize(pr.offset, bufferSize, pr.Size)
	chunk := getChunkByStartOffset(pr.offset, pr.Size)

	// fmt.Printf("Read Request: prOffset=%d, chunkNumber=%d, Range=%d-%d, requestedReadSize=%d\n",
	// 	pr.offset, chunk.number, chunk.startOffset, chunk.endOffset-1, requestedReadSize)

	cachedData, ok := pr.cache.Get(chunk.number)
	if ok {
		// fmt.Printf("Cache Hit: chunkNumber=%d\n", chunk.number)

		readBytes, readErr := pr.readFromCache(buffer, requestedReadSize, chunk, cachedData.([]byte))
		if readErr != nil {
			return readBytes, readErr
		}

		pr.offset += int64(readBytes)
		// fmt.Printf("After Read: prOffset=%d\n", pr.offset)
		return readBytes, nil
	}

	// fmt.Printf("Cache Miss: chunkNumber=%d, Fetching from server\n", chunk.number)

	readBytes, readErr := pr.fetchAndCacheChunk(buffer, requestedReadSize, chunk)
	if readErr != nil {
		return readBytes, readErr
	}

	pr.offset += int64(readBytes)
	// fmt.Printf("After Fetch and Read: prOffset=%d\n", pr.offset)
	return readBytes, nil
}

// Seek implements the io.Seeker interface
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

		// fmt.Printf("[Y] Seeked to end: prOffset=%d\n", pr.offset)

		return pr.offset, io.EOF
	}

	pr.offset = newOffset

	// fmt.Printf("[N] Seeked to: prOffset=%d\n", pr.offset)

	return pr.offset, nil
}

// Close implements the io.Closer interface
func (pr *PartialReader) Close() error {
	pr.cache.Purge()

	fmt.Println("Cache purged and PartialReader closed.")

	return nil
}

// checkIsMKV checks if the stream is an MKV file by reading the first few bytes
// func (pr *PartialReader) CheckIsMKV() error {
// 	rangeHeader := fmt.Sprintf("bytes=%d-%d", 0, 3)
// 	req, _ := http.NewRequest("GET", pr.url, nil)
// 	req.Header.Set("Range", rangeHeader)
//
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()
//
// 	// Read the first 4 bytes
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return err
// 	}
//
// 	// MKV magic number: 0x1A 0x45 0xDF 0xA3
// 	mkvSignature := []byte{0x1A, 0x45, 0xDF, 0xA3}
// 	header := body[:4]
//
// 	fmt.Println("Header:", header)
// 	fmt.Println("MKV Signature:", mkvSignature)
//
// 	// Check if the header matches the MKV signature
// 	pr.IsMkv = bytes.Equal(header, mkvSignature)
//
// 	return nil
// }

func calculateReadSize(offset int64, bufferSize int64, fileSize int64) int64 {
	if offset+bufferSize > fileSize {
		// Adjust the size to avoid reading past the end of the file
		return fileSize - offset
	}

	return bufferSize
}
