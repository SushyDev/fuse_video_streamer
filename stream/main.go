package stream

import (
	"context"
	"fmt"
	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"
	"fuse_video_streamer/stream/connection"
	"fuse_video_streamer/stream/transfer"
	"sync"
	"sync/atomic"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"
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
	id   string
	url  string
	size int64

	buffer ring_buffer.LockingRingBufferInterface

	ctx    context.Context
	cancel context.CancelFunc

	transfer *transfer.Transfer

	mu sync.Mutex

	closed atomic.Bool
}

func calculateBufferSize(fileSize int64) int64 {
	return min(fileSize, SmallVideoBuffer)

	switch {
	case fileSize < 1024*1024*1024: // < 1GB
		return SmallVideoBuffer
	case fileSize < 10*1024*1024*1024: // < 10GB
		return MediumVideoBuffer
	case fileSize < 50*1024*1024*1024: // < 50GB
		return LargeVideoBuffer
	default:
		return MaxBufferSize
	}
}

func calculatePreloadSize(bufferSize int64) int64 {
	return min(bufferSize/2, MaxPreloadSize)

	switch {
	case bufferSize <= SmallVideoBuffer:
		return SmallVideoPreloadSize
	case bufferSize <= MediumVideoBuffer:
		return MediumVideoPreloadSize
	case bufferSize <= LargeVideoBuffer:
		return LargeVideoPreloadSize
	default:
		return MaxPreloadSize
	}
}

func New(url string, size int64) (*Stream, error) {
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	bufferSize := calculateBufferSize(int64(size))

	buffer := ring_buffer.NewLockingRingBuffer(bufferSize, 0)

	ctx, cancel := context.WithCancel(context.Background())

	stream := &Stream{
		id: id,

		size: size,
		url:  url,

		buffer: buffer,

		ctx:    ctx,
		cancel: cancel,
	}

	return stream, nil
}

func (stream *Stream) Id() string {
	return stream.id
}

func (stream *Stream) ReadAt(p []byte, seekPosition int64) (int, error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if stream.isClosed() {
		return 0, fmt.Errorf("stream is closed")
	}

	if stream.buffer == nil {
		return 0, fmt.Errorf("buffer is closed")
	}

	requestedBytes := int64(len(p))

	if !stream.buffer.IsPositionAvailable(seekPosition) {
		err := stream.newTransfer(seekPosition)
		if err != nil {
			return 0, err
		}
	}

	requestedPosition := min(seekPosition+requestedBytes, stream.size)

	if !stream.buffer.IsPositionAvailable(requestedPosition) {
		ctx, cancel := context.WithTimeout(stream.ctx, 10*time.Second)
		defer cancel()

		ok := stream.buffer.WaitForPosition(ctx, requestedPosition)
		if !ok && !stream.isClosed() {
			return 0, fmt.Errorf("timeout waiting for the buffer to fill")
		}
	}

	return stream.buffer.ReadAt(p, seekPosition)
}

func (stream *Stream) Close() error {
	if !stream.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	stream.cancel()

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

func (stream *Stream) isClosed() bool {
	return stream.closed.Load()
}

func (stream *Stream) newTransfer(startPosition int64) error {
	if stream.isClosed() {
		return fmt.Errorf("stream is closed")
	}

	if stream.buffer == nil {
		return fmt.Errorf("buffer is closed")
	}

	if stream.transfer != nil {
		stream.transfer.Close()
		stream.transfer = nil
	}

	bufferSize := calculateBufferSize(stream.size)
	preloadSize := calculatePreloadSize(bufferSize)

	streamStartPosition := max(0, startPosition-preloadSize)

	connection, err := connection.NewConnection(stream.url, streamStartPosition)
	if err != nil {
		return err
	}

	stream.buffer.ResetToPosition(streamStartPosition)

	debugger := metrics.GetMetricsCollection()

	streamMetrics := debugger.NewStreamTransferMetrics(stream.id, stream.url, stream.size)

	transfer := transfer.NewTransfer(stream.buffer, connection, streamMetrics)
	stream.transfer = transfer

	return nil
}
