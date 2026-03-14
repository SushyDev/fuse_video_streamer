package streamable

import (
	"context"
	"fmt"
	"sync"
	"testing"

	mock_pool "fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool/mock"
	interfaces_logger "fuse_video_streamer/logger/interfaces"
	mock_logger "fuse_video_streamer/logger/mock"
	interfaces_stream "fuse_video_streamer/stream/interfaces"
	mock_stream "fuse_video_streamer/stream/mock"
)

// mockStreamFactory is an injectable StreamFactory for service tests.
type mockStreamFactory struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (factory *mockStreamFactory) NewStream(_ context.Context, _ uint64, _ uint64) (interfaces_stream.Stream, error) {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	if factory.err != nil {
		return nil, factory.err
	}
	factory.calls++
	return mock_stream.New(int64(factory.calls), 1024, "http://example.com"), nil
}

func (factory *mockStreamFactory) Close() error {
	return nil
}

// mockLoggerFactory satisfies interfaces_logger.LoggerFactory using the shared
// NoopLogger from the mock package.
type mockLoggerFactory struct{}

func (mockLoggerFactory) NewLogger(_ string) (interfaces_logger.Logger, error) {
	return mock_logger.NoopLogger{}, nil
}

var _ interfaces_logger.LoggerFactory = mockLoggerFactory{}

func newTestService(factory *mockStreamFactory) *Service {
	bufferPool := &mock_pool.BufferPool{}
	node := &stubStreamableNode{} // reuse from main_test.go (same package)
	logger := mock_logger.NoopLogger{}
	loggerFactory := mockLoggerFactory{}
	return NewService(node, nil, loggerFactory, factory, bufferPool, logger)
}

// TestNewHandle_Concurrent exercises concurrent NewHandle calls on a single
// Service. Any race on internal state will be caught by the race detector.
func TestNewHandle_Concurrent(t *testing.T) {
	t.Parallel()

	factory := &mockStreamFactory{}
	service := newTestService(factory)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			handle, err := service.NewHandle(context.Background())
			if err != nil {
				return
			}
			if handle != nil {
				handle.Close()
			}
		}()
	}

	wg.Wait()
}

// TestClose_WhileNewHandleInFlight calls service.Close while NewHandle is
// executing. Neither should panic or deadlock.
func TestClose_WhileNewHandleInFlight(t *testing.T) {
	t.Parallel()

	factory := &mockStreamFactory{}
	service := newTestService(factory)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			handle, err := service.NewHandle(context.Background())
			if err != nil || handle == nil {
				return
			}
			handle.Close()
		}
	}()

	go func() {
		defer wg.Done()
		service.Close()
	}()

	wg.Wait()
}

// TestNewHandle_AfterClose verifies that NewHandle after Close returns nil, nil
// without panicking.
func TestNewHandle_AfterClose(t *testing.T) {
	t.Parallel()

	factory := &mockStreamFactory{}
	service := newTestService(factory)

	service.Close()

	handle, err := service.NewHandle(context.Background())
	if err != nil {
		t.Fatalf("unexpected error after Close: %v", err)
	}
	if handle != nil {
		t.Fatal("expected nil handle after service is closed")
	}
}

// TestNewHandle_StreamFactoryError verifies that errors from the stream factory
// are propagated.
func TestNewHandle_StreamFactoryError(t *testing.T) {
	t.Parallel()

	factory := &mockStreamFactory{err: fmt.Errorf("stream unavailable")}
	service := newTestService(factory)

	_, err := service.NewHandle(context.Background())
	if err == nil {
		t.Fatal("expected error from factory, got nil")
	}
}
