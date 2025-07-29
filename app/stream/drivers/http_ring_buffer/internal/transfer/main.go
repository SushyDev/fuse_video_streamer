package transfer

import (
	"context"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	ring_buffer "github.com/sushydev/ring_buffer_go"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/filesystem/driver/provider/fuse/metrics"
	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/connection"
)

type Transfer struct {
	buffer     io.WriteCloser
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	metrics *metrics.StreamTransferMetrics
	logger  interfaces_logger.Logger

	wg *sync.WaitGroup

	closed atomic.Bool
}

var _ io.Closer = &Transfer{}

// Buffer pool for efficient memory reuse
var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 64*1024) // 64KB buffers
	},
}

func NewTransfer(buffer ring_buffer.LockingRingBufferInterface, connection *connection.Connection, metrics *metrics.StreamTransferMetrics, logger interfaces_logger.Logger) *Transfer {
	ctx, cancel := context.WithCancel(context.Background())

	transfer := &Transfer{
		buffer:     buffer,
		connection: connection,

		context: ctx,
		cancel:  cancel,

		wg: &sync.WaitGroup{},

		metrics: metrics,
		logger:  logger,
	}

	go transfer.start()

	return transfer
}

func (transfer *Transfer) start() {
	transfer.wg.Add(1)
	defer transfer.wg.Done()

	done := make(chan error, 1)

	go transfer.copyData(done)

	select {
	case <-transfer.context.Done():
		if transfer.connection != nil {
			transfer.connection.Close()
		}
	case err := <-done:
		switch err {
		case io.EOF:
			break
		case context.Canceled:
			break
		case nil:
			break
		default:
			if strings.HasPrefix(err.Error(), "Buffer is closed") {
				break
			}
			transfer.logger.Error("Error copying from connection", err)
		}
	}

	transfer.buffer.Write(ring_buffer.EOFMarker)
}

func (transfer *Transfer) copyData(done chan<- error) {
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		select {
		case <-transfer.context.Done():
			done <- context.Canceled
			return
		default:
		}

		bytesRead, readErr := transfer.connection.Read(buf)

		if bytesRead > 0 {
			_, writeErr := transfer.buffer.Write(buf[:bytesRead])
			if writeErr != nil {
				transfer.metrics.RecordTransferOperation(int64(bytesRead), true)
				done <- writeErr
				return
			}

			transfer.metrics.RecordTransferOperation(int64(bytesRead), false)
		}

		if readErr != nil {
			transfer.metrics.RecordTransferOperation(0, true)
			done <- readErr
			return
		}
	}
}

func (transfer *Transfer) Close() error {
	if !transfer.closed.CompareAndSwap(false, true) {
		return nil
	}

	if transfer.connection != nil {
		err := transfer.connection.Close()
		if err != nil {
			transfer.logger.Error("Error closing connection", err)
		}
	}

	transfer.cancel()

	transfer.wg.Wait()

	transfer.metrics.Finish()

	return nil
}
