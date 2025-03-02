package transfer

import (
	"context"
	"fmt"
	"fuse_video_steamer/stream/connection"
	"io"
	"sync"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

var _ io.Closer = &Transfer{}

type Transfer struct {
	buffer     io.WriteCloser
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	wg *sync.WaitGroup
}

func NewTransfer(buffer ring_buffer.LockingRingBufferInterface, connection *connection.Connection) *Transfer {
	ctx, cancel := context.WithCancel(context.Background())

	transfer := &Transfer{
		buffer:     buffer,
		connection: connection,
		context:    ctx,
		cancel:     cancel,
		wg:         &sync.WaitGroup{},
	}

	go transfer.start()

	return transfer
}

func (transfer *Transfer) start() {
	transfer.wg.Add(1)
	defer transfer.wg.Done()

	done := make(chan error, 1)

	go func() {
		_, err := io.Copy(transfer.buffer, transfer.connection)
		done <- err
	}()

	select {
	case <-transfer.context.Done():
		if transfer.connection != nil {
			transfer.connection.Close()
		}
	case err := <-done:
		switch err {
		case context.Canceled:
		case nil:
		default:
			fmt.Println("Error copying from connection:", err)
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
