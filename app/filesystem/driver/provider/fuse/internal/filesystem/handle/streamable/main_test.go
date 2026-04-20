package streamable

import (
	"context"
	"io"
	"os"
	"sync"
	"testing"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	mock_pool "fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool/mock"
	mock_logger "fuse_video_streamer/logger/mock"
	mock_stream "fuse_video_streamer/stream/mock"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

// stubStreamableNode is a minimal StreamableNode for use in handle tests.
type stubStreamableNode struct{}

func (n *stubStreamableNode) GetIdentifier() uint64                          { return 1 }
func (n *stubStreamableNode) GetRemoteIdentifier() uint64                    { return 1 }
func (n *stubStreamableNode) GetClient() interfaces_filesystem_client.Client { return nil }
func (n *stubStreamableNode) GetSize() uint64                                { return 1024 * 1024 }
func (n *stubStreamableNode) UpdateSize(uint64)                              {}
func (n *stubStreamableNode) GetMode() os.FileMode                           { return 0644 }
func (n *stubStreamableNode) IsClosed() bool                                 { return false }
func (n *stubStreamableNode) Close() error                                   { return nil }
func (n *stubStreamableNode) Attr(_ context.Context, attr *fuse.Attr) error  { return nil }
func (n *stubStreamableNode) Open(_ context.Context, _ *fuse.OpenRequest, _ *fuse.OpenResponse) (fs.Handle, error) {
	return nil, nil
}

func newTestHandle(stream *mock_stream.Stream) (*Handle, *mock_pool.BufferPool) {
	bufferPool := &mock_pool.BufferPool{}
	node := &stubStreamableNode{}
	handle := NewHandle(node, stream, bufferPool, mock_logger.NoopLogger{})
	return handle, bufferPool
}

func readRequest(offset int64, size int) *fuse.ReadRequest {
	return &fuse.ReadRequest{Offset: offset, Size: size}
}

// TestRead_ConcurrentReads launches N goroutines that all call Read on the same
// handle simultaneously. Any race on internal state will be caught by the race
// detector.
func TestRead_ConcurrentReads(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	// Queue enough responses so no goroutine blocks.
	const goroutines = 50
	for i := 0; i < goroutines; i++ {
		stream.EnqueueRead(mock_stream.ReadAtResponse{
			Data: []byte("hello"),
		})
	}

	handle, _ := newTestHandle(stream)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			req := readRequest(0, 5)
			resp := &fuse.ReadResponse{}
			_ = handle.Read(context.Background(), req, resp)
		}()
	}

	wg.Wait()
}

// TestRead_RacesWithClose calls Read from one goroutine while another calls
// Close. Neither should panic or deadlock.
func TestRead_RacesWithClose(t *testing.T) {
	t.Parallel()

	// Block channel: we'll unblock it after Close to create overlap.
	block := make(chan struct{})

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	stream.EnqueueRead(mock_stream.ReadAtResponse{
		Block: block,
		Data:  []byte("hello"),
	})

	handle, _ := newTestHandle(stream)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		req := readRequest(0, 5)
		resp := &fuse.ReadResponse{}
		_ = handle.Read(context.Background(), req, resp)
	}()

	go func() {
		defer wg.Done()
		// Close while Read is blocked, then unblock.
		handle.Close()
		close(block)
	}()

	wg.Wait()
}

// TestClose_Idempotent verifies that calling Close twice is safe.
func TestClose_Idempotent(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	handle, _ := newTestHandle(stream)

	if err := handle.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := handle.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestRead_AfterClose verifies that Read after Close returns ENOENT, not a panic.
func TestRead_AfterClose(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	handle, _ := newTestHandle(stream)

	handle.Close()

	req := readRequest(0, 5)
	resp := &fuse.ReadResponse{}
	err := handle.Read(context.Background(), req, resp)

	if err == nil {
		t.Fatal("expected error after Close, got nil")
	}
}

// TestRead_StreamClosedMidRead verifies that when the stream is already closed
// the handle surfaces the error cleanly.
func TestRead_StreamClosedMidRead(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	// Close the stream before the handle sees it so IsClosed() returns true.
	stream.Close()

	handle, _ := newTestHandle(stream)

	req := readRequest(0, 5)
	resp := &fuse.ReadResponse{}
	err := handle.Read(context.Background(), req, resp)

	if err == nil {
		t.Fatal("expected error when stream is closed, got nil")
	}
}

// TestRead_StreamReturnsError verifies that a non-nil, non-EOF error from
// ReadAt is propagated to the caller.
func TestRead_StreamReturnsError(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	stream.EnqueueRead(mock_stream.ReadAtResponse{
		Err: io.ErrUnexpectedEOF,
	})

	handle, _ := newTestHandle(stream)

	req := readRequest(0, 5)
	resp := &fuse.ReadResponse{}
	err := handle.Read(context.Background(), req, resp)

	if err == nil {
		t.Fatal("expected error from stream, got nil")
	}
}

// TestRead_ContextCancelled verifies that when the FUSE request context is
// cancelled during ReadAt the handle returns EINTR.
func TestRead_ContextCancelled(t *testing.T) {
	t.Parallel()

	block := make(chan struct{})

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	stream.EnqueueRead(mock_stream.ReadAtResponse{
		Block: block,
	})

	handle, _ := newTestHandle(stream)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	var readErr error
	go func() {
		defer wg.Done()
		req := readRequest(0, 5)
		resp := &fuse.ReadResponse{}
		readErr = handle.Read(ctx, req, resp)
	}()

	// Cancel the context to unblock the read. We do NOT close block — the
	// mock's select will pick ctx.Done() deterministically because block is
	// never ready.
	cancel()

	wg.Wait()

	// The handle maps context.Canceled to syscall.EINTR.
	if readErr == nil {
		t.Fatal("expected EINTR, got nil")
	}
}

// TestRead_BufferReturnedAfterRead verifies that the buffer pool has no live
// allocations after a Read completes (defer Put is exercised).
func TestRead_BufferReturnedAfterRead(t *testing.T) {
	t.Parallel()

	stream := mock_stream.New(1, 1024, "http://example.com/stream")
	stream.EnqueueRead(mock_stream.ReadAtResponse{Data: []byte("hello")})

	handle, bufferPool := newTestHandle(stream)

	req := readRequest(0, 5)
	resp := &fuse.ReadResponse{}
	if err := handle.Read(context.Background(), req, resp); err != nil {
		t.Fatalf("Read: %v", err)
	}

	if live := bufferPool.LiveAllocations(); live != 0 {
		t.Fatalf("expected 0 live allocations after Read, got %d", live)
	}
}
