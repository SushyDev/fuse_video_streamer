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
// It's 10% of the fileSize with a max of 128 MB and a minimum of 1 MB.
func calculateBufferSize(fileSize int64) int64 {
	return max(1*1024*1024, min(128*1024*1024, fileSize/10))
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

	if config.EnableDiskCache {
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

	// Close the buffer first to unblock any Transfer.copyData goroutine that
	// may be stuck in buffer.Write waiting for free space. Without this, the
	// transfer's WaitGroup.Wait in Transfer.Close would deadlock because the
	// writer goroutine never returns.
	stream.mu.Lock()
	buffer := stream.buffer
	stream.buffer = nil
	t := stream.transfer
	stream.transfer = nil
	stream.mu.Unlock()

	if buffer != nil {
		if err := buffer.Close(); err != nil {
			stream.logger.Error("error closing buffer", err)
		}
	}

	if t != nil {
		if err := t.Close(); err != nil {
			stream.logger.Error("error closing transfer", err)
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

	// Use a merged context: cancelled if either the per-request FUSE context
	// or the stream-lifetime context is cancelled. This ensures that if the
	// FUSE client abandons a read, WaitForPosition unblocks promptly.
	mergedCtx, mergedCancel := context.WithCancel(ctx)
	stop := context.AfterFunc(stream.ctx, mergedCancel)
	defer stop()
	defer mergedCancel()

	// Serialise all reads: the ring buffer has a monotonically advancing read
	// cursor (lastReadPosition), so concurrent ReadAt calls would corrupt each
	// other's position accounting. Hold the write lock for the entire
	// WaitForPosition + ReadAt sequence.
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.buffer == nil {
		return 0, fmt.Errorf("buffer is closed")
	}

	if stream.transfer == nil || !stream.buffer.IsPositionInCapacity(seekPosition, 0) {
		if err := stream.newTransferLocked(seekPosition); err != nil {
			return 0, fmt.Errorf("error before read at: %v", err)
		}
		// Re-read buffer reference after newTransfer (buffer is the same object,
		// but transfer may have changed). The buffer itself is not replaced.
	}

	buf := stream.buffer

	end := min(seekPosition+int64(len(p)), stream.size)
	if end > 0 {
		// Release the stream lock while waiting so that Close() and other
		// operations are not blocked for the duration of the network wait.
		stream.mu.Unlock()
		ok := buf.WaitForPosition(mergedCtx, end)
		stream.mu.Lock()

		if stream.buffer == nil {
			return 0, fmt.Errorf("buffer is closed")
		}

		if !ok && !buf.IsPositionAvailable(end) {
			if mergedCtx.Err() != nil {
				return 0, mergedCtx.Err()
			}
			// The position is no longer reachable in the current buffer window
			// (the read cursor has already advanced past it). Start a new transfer
			// from seekPosition and wait again.
			if err := stream.newTransferLocked(seekPosition); err != nil {
				return 0, fmt.Errorf("error restarting transfer: %v", err)
			}
			buf = stream.buffer
			stream.mu.Unlock()
			ok = buf.WaitForPosition(mergedCtx, end)
			stream.mu.Lock()
			if stream.buffer == nil {
				return 0, fmt.Errorf("buffer is closed")
			}
			if !ok {
				if mergedCtx.Err() != nil {
					return 0, mergedCtx.Err()
				}
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
	if stream.IsClosed() {
		return fmt.Errorf("stream is closed")
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	return stream.newTransferLocked(position)
}

// newTransferLocked creates a new transfer. Caller must hold stream.mu (write lock).
func (stream *Stream) newTransferLocked(seekPosition int64) error {
	if stream.buffer == nil {
		return fmt.Errorf("buffer is closed")
	}

	// Double-check under lock: another goroutine may have already created
	// a suitable transfer while we were waiting for the lock.
	if stream.transfer != nil && stream.buffer.IsPositionInCapacity(seekPosition, 0) {
		return nil
	}

	// Close the old buffer first to unblock any Transfer.copyData goroutine
	// that may be stuck in buffer.Write waiting for free space. Without this,
	// transfer.Close below would deadlock on wg.Wait.
	oldBuffer := stream.buffer
	oldBuffer.Close()

	if stream.transfer != nil {
		stream.transfer.Close()
		stream.transfer = nil
	}

	// Create a fresh buffer so the new transfer starts with clean state.
	bufferSize := calculateBufferSize(stream.size)
	stream.buffer = ring_buffer.NewLockingRingBuffer(bufferSize, seekPosition)

	conn, err := connection.NewConnection(stream.url, seekPosition)
	if err != nil {
		return err
	}

	logger, err := stream.loggerFactory.NewLogger("Stream Transfer")
	if err != nil {
		return err
	}

	t := transfer.NewTransfer(stream.buffer, conn, logger)
	stream.transfer = t

	return nil
}
