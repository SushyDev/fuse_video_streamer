package transfer

import (
	"context"
	"fmt"
	"fuse_video_streamer/logger"
	"fuse_video_streamer/stream/connection"
	"io"
	"strings"
	"sync"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

var _ io.Closer = &Transfer{}

type Transfer struct {
	buffer     io.WriteCloser
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	logger *logger.Logger

	wg *sync.WaitGroup
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

	done := make(chan error, 1)

	go func() {
		buf := make([]byte, 128*1024*1024) // 128MB buffer size
		_, err := io.CopyBuffer(transfer.buffer, transfer.connection, buf)
		done <- err
	}()

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

func (transfer *Transfer) Close() error {
	if transfer.isClosed() {
		return nil
	}

	if transfer.connection != nil {
		err := transfer.connection.Close()
		if err != nil {
			fmt.Println("Error closing connection:", err)
		}

		transfer.connection = nil
	}

	transfer.cancel()

	transfer.wg.Wait()

	return nil
}

func (transfer *Transfer) isClosed() bool {
	select {
	case <-transfer.context.Done():
		return true
	default:
		return false
	}
}
