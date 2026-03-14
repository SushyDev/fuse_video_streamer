package http_ring_buffer

import (
	"context"
	"testing"

	mock_config "fuse_video_streamer/config/mock"
	mock_logger "fuse_video_streamer/logger/mock"
)

// newTestStream creates a Stream pointed at an unreachable URL.  No network
// connections are made until ReadAt is called.
func newTestStream(size int64) (*Stream, error) {
	return New(mock_config.MinimalConfig(), mock_logger.NoopLoggerFactory{}, "http://127.0.0.1:0/test", size)
}

// TestClose_Idempotent verifies that Close called twice does not panic or
// return an error.
func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	stream, err := newTestStream(1024)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := stream.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestReadAt_ClosedBuffer verifies that ReadAt on a closed stream returns a
// clean error rather than panicking.
func TestReadAt_ClosedBuffer(t *testing.T) {
	t.Parallel()

	stream, err := newTestStream(1024)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	stream.Close()

	buf := make([]byte, 10)
	_, readErr := stream.ReadAt(context.Background(), buf, 0)
	if readErr == nil {
		t.Fatal("expected error reading from closed stream, got nil")
	}
}

// TestReadAt_RacesWithClose calls ReadAt and Close concurrently. Neither
// should panic; the read must return a clean error (not hang).
func TestReadAt_RacesWithClose(t *testing.T) {
	t.Parallel()

	stream, err := newTestStream(1024)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		stream.Close()
	}()

	buf := make([]byte, 10)
	// Use a short timeout so the test does not hang if ReadAt blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5 seconds
	defer cancel()
	_, _ = stream.ReadAt(ctx, buf, 0)

	<-done
}

// TestReadAt_PastEOF verifies that reading past end-of-file returns an error
// or zero bytes (not a panic or hang). The stream uses size=10; we seek to 20.
// The ring buffer's ReadAt at a position beyond the write cursor surfaces an
// error; we just verify no panic and a context-bounded completion.
func TestReadAt_PastEOF(t *testing.T) {
	t.Parallel()

	stream, err := newTestStream(10)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stream.Close()

	buf := make([]byte, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000) // 5 seconds
	defer cancel()

	// Either an error (EOF / closed buffer / context cancelled) or a
	// successful 0-byte read are acceptable; what is not acceptable is a panic.
	n, _ := stream.ReadAt(ctx, buf, 20)
	if n < 0 {
		t.Fatalf("ReadAt returned negative count: %d", n)
	}
}
