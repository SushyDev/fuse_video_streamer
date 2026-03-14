package factory

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	mock_config "fuse_video_streamer/config/mock"
	mock_client "fuse_video_streamer/filesystem/client/mock"
	mock_logger "fuse_video_streamer/logger/mock"
)

// immediateTimer replaces time.After with a channel that fires right away so
// retry-loop tests do not have to wait for real backoff delays.
func immediateTimer(_ time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

// buildFactory constructs a Factory from a mock FileSystem's GetStreamUrl func.
// The injected timer fires immediately so retry tests complete in microseconds.
func buildFactory(getStreamURL mock_client.GetStreamURLFunc) *Factory {
	rootNode := mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0)
	child := mock_client.NewNode(42, "file.mkv", fs.FileMode(0644), true, 1024*1024)
	children := map[uint64][]*mock_client.Node{
		rootNode.GetId(): {child},
	}
	fileSystem := mock_client.NewFileSystem(rootNode, children, getStreamURL)
	client := mock_client.NewClient("test", "/tmp", fileSystem)
	factory := New(mock_config.MinimalConfig(), client, mock_logger.NoopLoggerFactory{})
	factory.newTimer = immediateTimer
	return factory
}

// TestGetStreamURL_Concurrent runs N goroutines calling getStreamURL simultaneously.
// The cached URL item must not be corrupted.
func TestGetStreamURL_Concurrent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var callCount int

	factory := buildFactory(func(_ uint64) (string, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return "http://example.com/stream", nil
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			url, err := factory.getStreamURL(context.Background(), 42)
			if err != nil {
				t.Errorf("getStreamURL: %v", err)
				return
			}
			if url == "" {
				t.Error("expected non-empty URL")
			}
		}()
	}

	wg.Wait()
}

// TestGetStreamURL_ContextCancelled verifies that a cancelled context stops
// retries immediately.
func TestGetStreamURL_ContextCancelled(t *testing.T) {
	t.Parallel()

	// Always return empty URL so the factory would keep retrying.
	factory := buildFactory(func(_ uint64) (string, error) {
		return "", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	_, err := factory.getStreamURL(ctx, 42)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// TestGetStreamURL_RetriesOnError verifies that transient (non-syscall.Errno)
// errors are retried and the factory eventually succeeds.
func TestGetStreamURL_RetriesOnError(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	calls := 0

	factory := buildFactory(func(_ uint64) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls < 3 {
			return "", fmt.Errorf("transient error")
		}
		return "http://example.com/stream", nil
	})

	url, err := factory.getStreamURL(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	mu.Lock()
	got := calls
	mu.Unlock()

	if got < 3 {
		t.Fatalf("expected at least 3 calls, got %d", got)
	}
}

// TestGetStreamURL_PermanentError verifies that a syscall.Errno error is not
// retried — it is returned immediately.
func TestGetStreamURL_PermanentError(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	calls := 0

	factory := buildFactory(func(_ uint64) (string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return "", syscall.ENOENT
	})

	_, err := factory.getStreamURL(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	mu.Lock()
	got := calls
	mu.Unlock()

	if got != 1 {
		t.Fatalf("permanent error should stop after 1 call, got %d calls", got)
	}
}

// TestGetStreamURL_ExhaustsRetries verifies that after maxRetries the factory
// returns syscall.EAGAIN.
func TestGetStreamURL_ExhaustsRetries(t *testing.T) {
	t.Parallel()

	// Always return empty URL; immediateTimer makes the backoff instantaneous.
	factory := buildFactory(func(_ uint64) (string, error) {
		return "", nil
	})

	_, err := factory.getStreamURL(context.Background(), 42)
	if err != syscall.EAGAIN {
		t.Fatalf("expected EAGAIN, got: %v", err)
	}
}
