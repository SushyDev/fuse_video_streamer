package transfer

// Tests for the Transfer type, focusing on the wg.Add(1)-before-goroutine fix
// that prevents a race between NewTransfer and an immediate Close().

import (
	"testing"

	"fuse_video_streamer/stream/drivers/http_ring_buffer/internal/connection"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

// noopLogger satisfies interfaces_logger.Logger without importing the package.
type noopLogger struct{}

func (noopLogger) Info(string)         {}
func (noopLogger) Warn(string)         {}
func (noopLogger) Error(string, error) {}
func (noopLogger) Fatal(string, error) {}
func (noopLogger) Debug(string)        {}

// TestNewTransfer_ImmediateClose verifies that calling Close() immediately after
// NewTransfer() does not panic or deadlock.  Before the wg.Add(1) fix, calling
// Close() before start() had a chance to run could cause wg.Wait() to return
// while start() later called wg.Done() on a counter of 0, which panics.
func TestNewTransfer_ImmediateClose(t *testing.T) {
	// NOT parallel — GetMetricsCollection is not concurrency-safe.
	buf := ring_buffer.NewLockingRingBuffer(64*1024, 0)
	defer buf.Close()

	// Use a connection to a URL that will fail fast (no server). The URL is
	// intentionally unreachable so the HTTP request either errors immediately
	// or gets cancelled before completing.
	conn, err := connection.NewConnection("http://127.0.0.1:0/test", 0)
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}

	// Run multiple times to maximise the chance of triggering the race.
	for i := 0; i < 50; i++ {
		transfer := NewTransfer(buf, conn, noopLogger{})

		// Close immediately — before start() may have executed.
		if err := transfer.Close(); err != nil {
			t.Fatalf("Close() returned error: %v", err)
		}
	}
}

// TestNewTransfer_CloseIsIdempotent verifies that calling Close() twice does not
// panic or return an error.
func TestNewTransfer_CloseIsIdempotent(t *testing.T) {
	// NOT parallel — GetMetricsCollection is not concurrency-safe.
	buf := ring_buffer.NewLockingRingBuffer(64*1024, 0)
	defer buf.Close()

	conn, err := connection.NewConnection("http://127.0.0.1:0/test", 0)
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}

	transfer := NewTransfer(buf, conn, noopLogger{})

	if err := transfer.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := transfer.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
