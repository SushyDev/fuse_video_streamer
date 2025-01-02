package loader

import (
	"context"
	"fmt"
	"io"
	"time"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

var instanceCount = 0

func Copy(dst ring_buffer.LockingRingBufferInterface, src io.ReadCloser) (written int64, err error) {
	chunk := make([]byte, 8192)

	return CopyBuffer(dst, src, chunk)
}

func CopyBuffer(dst ring_buffer.LockingRingBufferInterface, src io.ReadCloser, buf []byte) (int64, error) {
	var normalDelay time.Duration = 100 * time.Microsecond
	var retryDelay time.Duration = normalDelay

	instanceCount++

	startTime := time.Now()
	totalBytesTransfered := int64(0)

	defer func() {
		mbps := float64(totalBytesTransfered) / time.Since(startTime).Seconds() / 1024 / 1024

		fmt.Printf("Copier instance %d closed. MBPS: %.2f\n", instanceCount, mbps)

		src.Close()
	}()

	for {
		select {
		case <-time.After(retryDelay):
		}

		bytesToOverwrite := max(dst.GetBytesToOverwrite(), 0)
		chunkSizeToRead := min(uint64(len(buf)), bytesToOverwrite)

		if chunkSizeToRead == 0 {
			retryDelay = 100 * time.Microsecond // Retry after 100 milliseconds
			continue
		}

		bytesTransfered, err := io.CopyN(dst, src, int64(chunkSizeToRead))
		totalBytesTransfered += bytesTransfered

		switch {
		case err == io.ErrUnexpectedEOF:
			return 0, fmt.Errorf("Unexpected EOF, Bytes transfered: %d", bytesTransfered)
			// continue

		case err == io.EOF:
			fmt.Println("EOF")
			return 0, io.EOF

		case err == context.Canceled:
			fmt.Println("Context Canceled")
			return 0, context.Canceled

		case err != nil:
			fmt.Println("Error:", err)
			// TODO RETRY
			retryDelay = 5 * time.Second
			continue
		default:
			retryDelay = normalDelay
		}
	}
}

