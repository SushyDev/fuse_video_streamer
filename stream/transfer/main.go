package transfer

// transfer job
// takes in buffer
// takes in connection

// gets stopped on connection close
// gets flushed on context cancel

import (
	"context"
	"fmt"
	"fuse_video_steamer/stream/connection"
	"io"
	"sync"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

var _ io.Closer = &Transfer{}

type Transfer struct {
	buffer     ring_buffer.LockingRingBufferInterface
	connection *connection.Connection

	context context.Context
	cancel  context.CancelFunc

	wg *sync.WaitGroup
}

var normalDelay time.Duration = 1 * time.Millisecond
var errorDelay time.Duration = 1 * time.Second
var waitDelay time.Duration = 1 * time.Millisecond

var buf = make([]byte, 1024*1024*4)

func NewTransfer(buffer ring_buffer.LockingRingBufferInterface, connection *connection.Connection) *Transfer {
	context, cancel := context.WithCancel(context.Background())

	transfer := &Transfer{
		buffer:     buffer,
		connection: connection,

		context: context,
		cancel:  cancel,

		wg: &sync.WaitGroup{},
	}

	go transfer.start()

	return transfer
}


func (transfer *Transfer) start() {
	defer func() {
		if transfer.connection != nil && !transfer.connection.IsClosed() {
			transfer.connection.Close()
		}

		fmt.Println("stopped transfer")

		transfer.wg.Done()
	}()

	var retryDelay time.Duration = normalDelay

	transfer.wg.Add(1)

	for {
		select {
		case <-transfer.context.Done():
			return
		case <-time.After(retryDelay):
		}

		if transfer.connection.IsClosed() {
			return
		}

		if transfer.connection == nil {
			retryDelay = errorDelay
			continue
		}

		bytesToOverwrite := transfer.buffer.GetBytesToOverwrite()
		chunkSizeToRead := min(uint64(len(buf)), bytesToOverwrite)

		if chunkSizeToRead == 0 {
			retryDelay = waitDelay
			continue
		}

		n, err := io.CopyN(transfer.buffer, transfer.connection, int64(chunkSizeToRead))

		switch {
		case err == io.ErrUnexpectedEOF:
			fmt.Println("Unexpected EOF")
			return
		case err == context.Canceled:
			fmt.Println("Context Canceled")
			retryDelay = errorDelay
			break
		case err == io.EOF:
			fmt.Println("EOF")
			retryDelay = errorDelay
			break
		case err != nil:
			fmt.Println("Error:", err)
			return
		default:
			if n > 0 {
				retryDelay = normalDelay
			}

			break
		}
	}
}

func (transfer *Transfer) IsClosed() bool {
	select {
	case <-transfer.context.Done():
		return true
	default:
		return false
	}
}

func (transfer *Transfer) Close() error {
	if transfer.IsClosed() {
		return nil
	}

	fmt.Println("closing transfer")

	transfer.cancel()

	transfer.wg.Wait()

	fmt.Println("closed transfer")

	return nil
}
