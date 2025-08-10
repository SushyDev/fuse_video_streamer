// Package disk_cache provides functionality to manage a disk cache for HTTP streaming,
// now with a persistent .lock file that tracks written ranges for reliable detection.
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

	return &DiskCache{
		file:     file,
		lockFile: lockFile,
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

func readRanges(f *os.File) ([]Range, error) {
	_, _ = f.Seek(0, 0)
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

func writeRanges(f *os.File, ranges []Range) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, r := range ranges {
		fmt.Fprintf(w, "%d %d\n", r.Start, r.End)
	}
	return w.Flush()
}

func insertRange(ranges []Range, newRange Range) []Range {
	ranges = append(ranges, newRange)
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].Start < ranges[j].Start
	})

	merged := []Range{}
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

func (diskCache *DiskCache) recordRange(start, end int64) error {
	ranges, _ := readRanges(diskCache.lockFile)
	ranges = insertRange(ranges, Range{Start: start, End: end})
	return writeRanges(diskCache.lockFile, ranges)
}

// -------- Public Methods --------

func (diskCache *DiskCache) IsPositionOnDisk(start, end int64) (bool, error) {
	diskCache.mu.RLock()
	defer diskCache.mu.RUnlock()

	if diskCache.file == nil || diskCache.lockFile == nil {
		return false, os.ErrClosed
	}

	ranges, err := readRanges(diskCache.lockFile)
	if err != nil {
		return false, err
	}

	for _, r := range ranges {
		if r.Start <= start && r.End >= end {
			return true, nil
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

	if err := diskCache.file.Sync(); err != nil {
		return n, fmt.Errorf("failed to sync file after write: %w", err)
	}

	if err := diskCache.recordRange(seekPosition, seekPosition+int64(n)); err != nil {
		return n, fmt.Errorf("failed to update lockfile: %w", err)
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

	return nil
}

func (diskCache *DiskCache) IsClosed() bool {
	return diskCache.closed.Load()
}
