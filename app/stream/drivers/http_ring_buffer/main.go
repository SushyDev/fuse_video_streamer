package http_ring_buffer

import (
	"context"
	"fmt"
	//"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"

	interfaces_logger "fuse_video_streamer/logger/interfaces"
	interfaces_stream "fuse_video_streamer/stream/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"
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

	buffer ring_buffer.LockingRingBufferInterface
	diskCache *disk_cache.DiskCache

	ctx    context.Context
	cancel context.CancelFunc

	transfer *transfer.Transfer

	mu sync.Mutex

	closed atomic.Bool
}

var _ interfaces_stream.Stream = &Stream{}

func calculateBufferSize(fileSize int64) int64 {
	return min(fileSize, SmallVideoBuffer)
}

func calculatePreloadSize(bufferSize int64) int64 {
	return min(bufferSize/2, MaxPreloadSize)
}

func New(loggerFactory interfaces_logger.LoggerFactory, url string, size int64) (*Stream, error) {
	fmt.Printf("Creating new stream for URL: %s with size: %d bytes\n", url, size)

	identifier := time.Now().UnixNano()

	bufferSize := calculateBufferSize(int64(size))

	buffer := ring_buffer.NewLockingRingBuffer(bufferSize, 0)

	diskCache, err := disk_cache.NewDiskCache(url, size)
	if err != nil {
		return nil, fmt.Errorf("error creating disk cache: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())


	stream := &Stream{
		identifier: identifier,

		size: size,
		url:  url,

		loggerFactory: loggerFactory,

		buffer: buffer,
		diskCache: diskCache,

		ctx:    ctx,
		cancel: cancel,
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

func (stream *Stream) ReadAt(p []byte, seekPosition int64) (int, error) {
	if stream.IsClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	requestedBytes := int64(len(p))
	percentage := float64(requestedBytes) / float64(stream.size) * 100
	fmt.Printf("Reading %d bytes at position %d. percentage: %v\n", requestedBytes, seekPosition, percentage)

	if stream.diskCache != nil { 
		isPositionOnDisk, diskErr := stream.diskCache.IsPositionOnDisk(seekPosition, seekPosition+requestedBytes)
		if diskErr == nil && isPositionOnDisk {
			fmt.Printf("DISK %d bytes at position %d. percentage: %v\n", requestedBytes, seekPosition, percentage)
			return stream.diskCache.ReadAt(p, seekPosition)
		} else if (diskErr != nil) {
			fmt.Printf("Error checking disk cache: %v\n", diskErr)
		}
	}

	read, err := stream.readFromBuffer(p, seekPosition)
	if err != nil {
		return read, err
	}

	if read > 0 && stream.diskCache != nil {
		copyOfP := make([]byte, read)
		copy(copyOfP, p[:read])

		_, writeErr := stream.diskCache.WriteAt(copyOfP, seekPosition)
		if writeErr != nil {
			return read, fmt.Errorf("error writing to disk cache: %v", writeErr)
		}

		fmt.Printf("Wrote %d bytes to disk cache at position %d - %v percent \n", read, seekPosition, percentage)
	}

	return read, err
}

func (stream *Stream) Close() error {
	if !stream.closed.CompareAndSwap(false, true) {
		return nil
	}

	stream.cancel()

	// log trace golang debug
	//debug.PrintStack()

	if stream.diskCache != nil {
		fmt.Println("Closing disk cache...")
		err := stream.diskCache.Close()
		if err != nil {
			return fmt.Errorf("error closing disk cache: %v", err)
		}
		fmt.Println("Disk cache closed successfully.")

		stream.diskCache = nil
	}

	if stream.buffer != nil {
		err := stream.buffer.Close()
		if err != nil {
			return fmt.Errorf("error closing buffer: %v", err)
		}

		stream.buffer = nil
	}

	if stream.transfer != nil {
		err := stream.transfer.Close()
		if err != nil {
			fmt.Println("error closing transfer:", err)
		}

		stream.transfer = nil
	}

	return nil
}

func (stream *Stream) IsClosed() bool {
	return stream.closed.Load()
}

func (stream *Stream) readFromBuffer(p []byte, seekPosition int64) (int, error) {
	if stream.IsClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	if stream.buffer == nil {
		return 0, fmt.Errorf("buffer is closed")
	}

	requestedBytes := int64(len(p))
	requestedPosition := min(seekPosition+requestedBytes, stream.size)

	if !stream.buffer.IsPositionAvailable(seekPosition) {
		err := stream.newTransfer(seekPosition)
		if err != nil {
			return 0, err
		}
	}

	if !stream.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(stream.ctx, 2*time.Second)
		defer cancel()

		ok := stream.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok && !stream.IsClosed() {
			return 0, fmt.Errorf("timeout waiting for the buffer to fill")
		}
	}

	return stream.buffer.ReadAt(p, seekPosition)
}

func (stream *Stream) newTransfer(seekPosition int64) error {
	if stream.IsClosed() {
		return fmt.Errorf("stream is closed")
	}

	if stream.buffer == nil {
		return fmt.Errorf("buffer is closed")
	}

	fmt.Printf("Creating new transfer for stream %d at position %d at url %s\n", stream.identifier, seekPosition, stream.url)

	if stream.transfer != nil {
		stream.transfer.Close()
		stream.transfer = nil
	}

	bufferSize := calculateBufferSize(stream.size)
	preloadSize := calculatePreloadSize(bufferSize)

	streamStartPosition := max(0, seekPosition-preloadSize)

	connection, err := connection.NewConnection(stream.url, streamStartPosition)
	if err != nil {
		return err
	}

	stream.buffer.ResetToPosition(streamStartPosition)

	debugger := metrics.GetMetricsCollection()

	streamMetrics := debugger.NewStreamTransferMetrics(stream.identifier, stream.url, stream.size)

	logger, err := stream.loggerFactory.NewLogger("Stream Transfer")
	if err != nil {
		return err
	}

	transfer := transfer.NewTransfer(stream.buffer, connection, streamMetrics, logger)
	stream.transfer = transfer

	return nil
}
