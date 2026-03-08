// Package disk_cache provides functionality to manage a disk cache for HTTP streaming,
// with a persistent .lock file that tracks written ranges for reliable detection.
//
// Ranges are maintained in-memory as a sorted, merged interval list and flushed
// to the .lock file on mutations. This avoids O(n) file reads on every operation
// and eliminates file-cursor races that occurred under concurrent RLock access.
package disk_cache

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type Range struct {
	Start, End int64
}

type DiskCache struct {
	file     *os.File
	lockFile *os.File

	// ranges is the in-memory representation of written byte ranges.
	// It is always kept sorted and merged.
	ranges []Range

	mu sync.RWMutex

	closed atomic.Bool
}

func NewDiskCache(url string, size int64) (*DiskCache, error) {
	file, err := getFile(url, size)
	if err != nil {
		return nil, err
	}

	lockPath := getFilePath(url) + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Load existing ranges from the lockfile into memory.
	ranges, err := readRangesFromFile(lockFile)
	if err != nil {
		// Non-fatal: start with empty ranges if the lockfile is corrupt.
		ranges = nil
	}

	return &DiskCache{
		file:     file,
		lockFile: lockFile,
		ranges:   ranges,
	}, nil
}

func getFilePath(url string) string {
	if url == "" {
		return ".cache/empty.cache"
	}

	if err := os.MkdirAll(".cache", 0755); err != nil {
		return ".cache/error.cache"
	}

	parts := strings.Split(url, "/")
	fileName := parts[len(parts)-1]

	hash := sha256.Sum256([]byte(fileName))
	return ".cache/" + hex.EncodeToString(hash[:]) + ".cache"
}

func getFile(url string, size int64) (*os.File, error) {
	fileExists, err := fileExists(url)
	if err != nil {
		return nil, err
	}

	if fileExists {
		return fileOpen(url, size)
	}

	return fileCreate(url, size)
}

func fileExists(url string) (bool, error) {
	filePath := getFilePath(url)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func fileOpen(url string, size int64) (*os.File, error) {
	filePath := getFilePath(url)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	if err := file.Chmod(0666); err != nil {
		file.Close()
		return nil, err
	}

	if info.Size() < size {
		fmt.Printf("Resizing file %s from %d to %d bytes\n", filePath, info.Size(), size)
		if err := file.Truncate(size); err != nil {
			file.Close()
			return nil, err
		}
	}

	return file, nil
}

func fileCreate(url string, size int64) (*os.File, error) {
	filePath := getFilePath(url)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	if err := file.Chmod(0666); err != nil {
		file.Close()
		return nil, err
	}

	if err := file.Truncate(size); err != nil {
		file.Close()
		return nil, err
	}

	return file, nil
}

// -------- Lockfile Management --------

// readRangesFromFile reads range entries from a lockfile on disk.
// Only called once at startup.
func readRangesFromFile(f *os.File) ([]Range, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	var ranges []Range
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var s, e int64
		if _, err := fmt.Sscanf(scanner.Text(), "%d %d", &s, &e); err == nil {
			ranges = append(ranges, Range{Start: s, End: e})
		}
	}
	return ranges, scanner.Err()
}

// flushRangesToFile writes the in-memory ranges to the lockfile.
// Must be called under write lock.
func (diskCache *DiskCache) flushRangesToFile() error {
	if diskCache.lockFile == nil {
		return os.ErrClosed
	}
	if err := diskCache.lockFile.Truncate(0); err != nil {
		return err
	}
	if _, err := diskCache.lockFile.Seek(0, 0); err != nil {
		return err
	}
	w := bufio.NewWriter(diskCache.lockFile)
	for _, r := range diskCache.ranges {
		if _, err := fmt.Fprintf(w, "%d %d\n", r.Start, r.End); err != nil {
			return err
		}
	}
	return w.Flush()
}

func insertRange(ranges []Range, newRange Range) []Range {
	ranges = append(ranges, newRange)
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	var merged []Range
	for _, r := range ranges {
		if len(merged) == 0 {
			merged = append(merged, r)
			continue
		}
		last := &merged[len(merged)-1]
		// Merge if overlapping or touching
		if r.Start <= last.End {
			if r.End > last.End {
				last.End = r.End
			}
		} else {
			merged = append(merged, r)
		}
	}
	return merged
}

// -------- Public Methods --------

// IsPositionOnDisk checks whether the byte range [start, end] is fully covered
// by a previously written range. Uses the in-memory range list (no disk I/O).
func (diskCache *DiskCache) IsPositionOnDisk(start, end int64) (bool, error) {
	diskCache.mu.RLock()
	defer diskCache.mu.RUnlock()

	if diskCache.closed.Load() {
		return false, os.ErrClosed
	}

	for _, r := range diskCache.ranges {
		if r.Start <= start && r.End >= end {
			return true, nil
		}
		// Since ranges are sorted, if this range starts after our start
		// position, no subsequent range can cover [start, end].
		if r.Start > start {
			break
		}
	}
	return false, nil
}

func (diskCache *DiskCache) WriteAt(p []byte, seekPosition int64) (int, error) {
	if diskCache.IsClosed() {
		return 0, os.ErrClosed
	}

	diskCache.mu.Lock()
	defer diskCache.mu.Unlock()

	if diskCache.file == nil {
		return 0, os.ErrClosed
	}

	n, err := diskCache.file.WriteAt(p, seekPosition)
	if err != nil {
		return n, err
	}

	// Update in-memory ranges and flush to lockfile.
	diskCache.ranges = insertRange(diskCache.ranges, Range{Start: seekPosition, End: seekPosition + int64(n)})
	if flushErr := diskCache.flushRangesToFile(); flushErr != nil {
		return n, fmt.Errorf("failed to update lockfile: %w", flushErr)
	}

	return n, nil
}

func (diskCache *DiskCache) ReadAt(p []byte, seekPosition int64) (int, error) {
	if diskCache.IsClosed() {
		return 0, os.ErrClosed
	}

	diskCache.mu.RLock()
	defer diskCache.mu.RUnlock()

	if diskCache.file == nil {
		return 0, os.ErrClosed
	}

	return diskCache.file.ReadAt(p, seekPosition)
}

func (diskCache *DiskCache) Close() error {
	if !diskCache.closed.CompareAndSwap(false, true) {
		return nil
	}

	diskCache.mu.Lock()
	defer diskCache.mu.Unlock()

	if diskCache.file != nil {
		if err := diskCache.file.Close(); err != nil {
			return err
		}
		diskCache.file = nil
	}

	if diskCache.lockFile != nil {
		if err := diskCache.lockFile.Close(); err != nil {
			return err
		}
		diskCache.lockFile = nil
	}

	diskCache.ranges = nil

	return nil
}

func (diskCache *DiskCache) IsClosed() bool {
	return diskCache.closed.Load()
}
