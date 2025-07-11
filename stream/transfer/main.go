package transfer

import (
	"context"
	"fmt"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/stream/connection"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

type Transfer struct {
	buffer     io.WriteCloser
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	logger *logger.Logger

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

func NewTransfer(buffer ring_buffer.LockingRingBufferInterface, connection *connection.Connection) *Transfer {
	logger, err := logger.NewLogger("Transfer")
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	transfer := &Transfer{
		buffer:     buffer,
		connection: connection,

		context: ctx,
		cancel:  cancel,

		wg: &sync.WaitGroup{},

		logger: logger,
	}

	go transfer.start()

	return transfer
}

func (transfer *Transfer) start() {
	transfer.wg.Add(1)
	defer transfer.wg.Done()

	fmt.Println("Starting transfer...")

	done := make(chan error, 1)

	go transfer.copyData(done)

	select {
	case <-transfer.context.Done():
		if transfer.connection != nil {
			transfer.connection.Close()
			transfer.connection = nil
		}
	case err := <-done:
		switch err {
		case context.Canceled:
		case nil:
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
				done <- writeErr
				return
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				done <- nil
			} else {
				done <- readErr
			}

			return
		}
	}
}

func (transfer *Transfer) Close() error {
	if !transfer.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	fmt.Println("Closing transfer...")

	if transfer.connection != nil {
		err := transfer.connection.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}

	transfer.cancel()

	transfer.wg.Wait()
	
	fmt.Println("Transfer closed")

	return nil
}
