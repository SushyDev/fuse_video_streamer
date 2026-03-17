package transfer

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"

	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/connection"
)

type Transfer struct {
	buffer     io.WriteCloser
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	logger interfaces_logger.Logger

	wg *sync.WaitGroup

	closed atomic.Bool
}

// isBadContentLengthError checks if an error is due to a bad Content-Length header
func isBadContentLengthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "Content-Length")
}

var _ io.Closer = &Transfer{}

// Buffer pool for efficient memory reuse
var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, 64*1024) // 64KB buffers
	},
}

func NewTransfer(buffer ring_buffer.LockingRingBufferInterface, connection *connection.Connection, logger interfaces_logger.Logger) *Transfer {
	ctx, cancel := context.WithCancel(context.Background())

	wg := &sync.WaitGroup{}

	// Add to WaitGroup BEFORE launching the goroutine to prevent a race
	// where Close() calls wg.Wait() before start() has a chance to run.
	wg.Add(1)

	transfer := &Transfer{
		buffer:     buffer,
		connection: connection,

		context: ctx,
		cancel:  cancel,

		wg: wg,

		logger: logger,
	}

	go transfer.start()

	return transfer
}

func (transfer *Transfer) start() {
	defer transfer.wg.Done()

	done := make(chan error, 1)

	go transfer.copyData(done)

	select {
	case <-transfer.context.Done():
		// Close the connection to unblock any in-progress body.Read in copyData,
		// then wait for the copier to finish before marking EOF.
		if transfer.connection != nil {
			transfer.connection.Close()
		}
		err := <-done
		switch err {
		case io.EOF, context.Canceled, nil:
			break
		default:
			if errors.Is(err, os.ErrClosed) {
				break
			}
			transfer.logger.Error("Error copying from connection (context cancelled)", err)
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
			if errors.Is(err, os.ErrClosed) {
				break
			}

			// Log Content-Length errors separately - they're server-side issues
			if isBadContentLengthError(err) {
				transfer.logger.Error("Server returned invalid Content-Length header - stream interrupted", err)
			} else {
				transfer.logger.Error("Error copying from connection", err)
			}
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
				done <- writeErr
				return
			}
		}

		if readErr != nil {
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

	return nil
}
