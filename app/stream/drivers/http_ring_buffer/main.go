// Package http_ring_buffer implements a streaming solution using a ring buffer
package http_ring_buffer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"

	"fuse_video_streamer/config"
	interfaces_logger "fuse_video_streamer/logger/interfaces"
	interfaces_stream "fuse_video_streamer/stream/interfaces"

	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/connection"
	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/disk_cache"
	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/transfer"
)

const (
	SmallVideoBuffer  = int64(64 * 1024 * 1024)   // 64MB for < 1GB files
	MediumVideoBuffer = int64(256 * 1024 * 1024)  // 256MB for 1-10GB files
	LargeVideoBuffer  = int64(512 * 1024 * 1024)  // 512MB for 10GB+ files
	MaxBufferSize     = int64(1024 * 1024 * 1024) // 1GB absolute max
)

const (
	SmallVideoPreloadSize  = int64(32 * 1024 * 1024)  // 32MB for < 1GB files
	MediumVideoPreloadSize = int64(128 * 1024 * 1024) // 128MB for 1-10GB files
	LargeVideoPreloadSize  = int64(256 * 1024 * 1024) // 256MB for 10GB+ files
	MaxPreloadSize         = int64(16 * 1024 * 1024)  // 16MB absolute max preload size
)

type Stream struct {
	identifier int64
	url        string
	size       int64

	loggerFactory interfaces_logger.LoggerFactory

	buffer    ring_buffer.LockingRingBufferInterface
	diskCache *disk_cache.DiskCache

	ctx    context.Context
	cancel context.CancelFunc

	transfer *transfer.Transfer

	logger interfaces_logger.Logger

	mu sync.RWMutex

	closed atomic.Bool
}

var _ interfaces_stream.Stream = &Stream{}

// calculateBufferSize determines the buffer size based on the file size.
// Its 10% of the fileSize with a max of 128MB
func calculateBufferSize(fileSize int64) int64 {
	return min(128*1024*1024, fileSize/10)
}

func calculatePreloadSize(bufferSize int64) int64 {
	return bufferSize / 4
}

func New(config *config.Config, loggerFactory interfaces_logger.LoggerFactory, url string, size int64) (*Stream, error) {
	identifier := time.Now().UnixNano()

	bufferSize := calculateBufferSize(int64(size))

	buffer := ring_buffer.NewLockingRingBuffer(bufferSize, 0)

	logger, err := loggerFactory.NewLogger("Stream")
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	stream := &Stream{
		identifier: identifier,

		size: size,
		url:  url,

		loggerFactory: loggerFactory,

		buffer: buffer,

		ctx:    ctx,
		cancel: cancel,

		logger: logger,
	}

	if config.GetEnableDiskCache() {
		diskCache, err := disk_cache.NewDiskCache(url, size)
		if err != nil {
			return nil, fmt.Errorf("error creating disk cache: %v", err)
		}

		stream.diskCache = diskCache
	}

	return stream, nil
}

func (stream *Stream) Identifier() int64 {
	return stream.identifier
}

func (stream *Stream) Size() int64 {
	return stream.size
}

func (stream *Stream) URL() string {
	return stream.url
}

func (stream *Stream) ReadAt(ctx context.Context, p []byte, seekPosition int64) (int, error) {
	if stream.IsClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	requestedBytes := int64(len(p))

	if stream.diskCache != nil {
		isPositionOnDisk, diskErr := stream.diskCache.IsPositionOnDisk(seekPosition, seekPosition+requestedBytes)
		if diskErr == nil && isPositionOnDisk {
			return stream.diskCache.ReadAt(p, seekPosition)
		}
	}

	read, err := stream.readFromBuffer(ctx, p, seekPosition)
	if err != nil {
		return read, err
	}

	// Write-through to disk cache. Failures are logged but not propagated,
	// because the data was already successfully read from the ring buffer.
	if stream.diskCache != nil && read > 0 {
		copyOfP := make([]byte, read)
		copy(copyOfP, p[:read])

		_, writeErr := stream.diskCache.WriteAt(copyOfP, seekPosition)
		if writeErr != nil {
			stream.logger.Error("disk cache write failed (non-fatal)", writeErr)
		}
	}

	return read, err
}

func (stream *Stream) Close() error {
	if !stream.closed.CompareAndSwap(false, true) {
		return nil
	}

	stream.cancel()

	// Close transfer first so its goroutines stop writing to the buffer.
	stream.mu.Lock()
	t := stream.transfer
	stream.transfer = nil
	stream.mu.Unlock()

	if t != nil {
		if err := t.Close(); err != nil {
			stream.logger.Error("error closing transfer", err)
		}
	}

	if stream.buffer != nil {
		err := stream.buffer.Close()
		if err != nil {
			return fmt.Errorf("error closing buffer: %v", err)
		}
	}

	if stream.diskCache != nil {
		err := stream.diskCache.Close()
		if err != nil {
			return fmt.Errorf("error closing disk cache: %v", err)
		}
	}

	return nil
}

func (stream *Stream) IsClosed() bool {
	return stream.closed.Load()
}

func (stream *Stream) readFromBuffer(ctx context.Context, p []byte, seekPosition int64) (int, error) {
	if stream.IsClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	// Take a local reference to the buffer under read lock to avoid TOCTOU
	// races with Close() which may nil out stream.buffer.
	stream.mu.RLock()
	buf := stream.buffer
	currentTransfer := stream.transfer
	stream.mu.RUnlock()

	if buf == nil {
		return 0, fmt.Errorf("buffer is closed")
	}

	if currentTransfer == nil || !buf.IsPositionInCapacity(seekPosition, 16*1024*1024) {
		err := stream.newTransfer(seekPosition)
		if err != nil {
			return 0, fmt.Errorf("error before read at: %v", err)
		}
		// Re-read buffer reference after newTransfer (buffer is the same object,
		// but transfer may have changed). The buffer itself is not replaced.
	}

	// Use a merged context: cancelled if either the per-request FUSE context
	// or the stream-lifetime context is cancelled. This ensures that if the
	// FUSE client abandons a read, WaitForPosition unblocks promptly.
	mergedCtx, mergedCancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-stream.ctx.Done():
			mergedCancel()
		case <-mergedCtx.Done():
		}
	}()
	defer mergedCancel()

	end := min(seekPosition+int64(len(p)), stream.size)
	if end > 0 {
		ok := buf.WaitForPosition(mergedCtx, end)
		if !ok && !buf.IsPositionAvailable(end) {
			// Likely EOF before full span available, or context cancelled.
			if mergedCtx.Err() != nil {
				return 0, mergedCtx.Err()
			}
		}
	}

	return buf.ReadAt(p, seekPosition)
}

// SeekTo restarts the underlying HTTP transfer from the given byte position.
// It is safe for concurrent use. This is the mechanism by which a media
// player's backward-seek (which causes ErrOutOfRange from the ring buffer)
// is recovered without closing the stream entirely.
func (stream *Stream) SeekTo(position int64) error {
	return stream.newTransfer(position)
}

// newTransfer creates a new transfer for the stream at the specified seek position.
func (stream *Stream) newTransfer(seekPosition int64) error {
	if stream.IsClosed() {
		return fmt.Errorf("stream is closed")
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.buffer == nil {
		return fmt.Errorf("buffer is closed")
	}

	// Double-check under lock: another goroutine may have already created
	// a suitable transfer while we were waiting for the lock.
	if stream.transfer != nil && stream.buffer.IsPositionInCapacity(seekPosition, 16*1024*1024) {
		return nil
	}

	if stream.transfer != nil {
		stream.transfer.Close()
		stream.transfer = nil
	}

	conn, err := connection.NewConnection(stream.url, seekPosition)
	if err != nil {
		return err
	}

	stream.buffer.ResetToPosition(seekPosition)

	logger, err := stream.loggerFactory.NewLogger("Stream Transfer")
	if err != nil {
		return err
	}

	t := transfer.NewTransfer(stream.buffer, conn, logger)
	stream.transfer = t

	return nil
}
