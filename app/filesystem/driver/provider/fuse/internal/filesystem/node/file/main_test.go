package file

import (
	"context"
	"sync"
	"testing"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	mock_logger "fuse_video_streamer/logger/mock"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

// stubFileHandle is a minimal FileHandle for testing the file node.
type stubFileHandle struct {
	id uint64
}

func (h *stubFileHandle) GetIdentifier() uint64 { return h.id }
func (h *stubFileHandle) Close() error          { return nil }
func (h *stubFileHandle) IsClosed() bool        { return false }

func (h *stubFileHandle) ReadAll(_ context.Context) ([]byte, error) { return nil, nil }
func (h *stubFileHandle) Read(_ context.Context, _ *fuse.ReadRequest, _ *fuse.ReadResponse) error {
	return nil
}
func (h *stubFileHandle) Write(_ context.Context, _ *fuse.WriteRequest, _ *fuse.WriteResponse) error {
	return nil
}
func (h *stubFileHandle) Release(_ context.Context, _ *fuse.ReleaseRequest) error { return nil }
func (h *stubFileHandle) Flush(_ context.Context, _ *fuse.FlushRequest) error     { return nil }
func (h *stubFileHandle) Fsync(_ context.Context, _ *fuse.FsyncRequest) error     { return nil }

var _ interfaces_handle.FileHandle = (*stubFileHandle)(nil)

// stubFileHandleService creates stubFileHandle instances.
type stubFileHandleService struct {
	mu      sync.Mutex
	counter uint64
	closed  bool
}

func (service *stubFileHandleService) NewHandle(_ interfaces_node.FileNode) (interfaces_handle.FileHandle, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.closed {
		return nil, nil
	}
	service.counter++
	return &stubFileHandle{id: service.counter}, nil
}

func (service *stubFileHandleService) Close() error {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.closed = true
	return nil
}

func (service *stubFileHandleService) IsClosed() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.closed
}

var _ interfaces_handle.FileHandleService = (*stubFileHandleService)(nil)

// newTestFileNode creates a Node wired with a stub handle service for testing.
func newTestFileNode() (*Node, *stubFileHandleService) {
	service := &stubFileHandleService{}
	node := NewNode(
		nil, // client — not exercised in these tests
		mock_logger.NoopLoggerFactory{},
		service,
		mock_logger.NoopLogger{},
		1,    // identifier
		100,  // remoteIdentifier
		1024, // size
		0644,
	)
	return node, service
}

func stubClient() interfaces_filesystem_client.Client {
	return nil
}

// openRequest returns a minimal fuse.OpenRequest.
func openRequest() *fuse.OpenRequest {
	return &fuse.OpenRequest{}
}

// TestOpen_Concurrent calls Open from N goroutines simultaneously.
// Any corruption of the handles slice is detected by the race detector.
func TestOpen_Concurrent(t *testing.T) {
	t.Parallel()

	node, _ := newTestFileNode()

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			resp := &fuse.OpenResponse{}
			handle, err := node.Open(context.Background(), openRequest(), resp)
			if err != nil {
				return
			}
			if h, ok := handle.(interfaces_handle.FileHandle); ok {
				h.Close()
			}
		}()
	}

	wg.Wait()
}

// TestClose_WhileOpenInFlight calls Close while Open goroutines are running.
func TestClose_WhileOpenInFlight(t *testing.T) {
	t.Parallel()

	node, _ := newTestFileNode()

	var wg sync.WaitGroup
	const openGoroutines = 50
	wg.Add(openGoroutines + 1)

	start := make(chan struct{})

	for i := 0; i < openGoroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			resp := &fuse.OpenResponse{}
			handle, err := node.Open(context.Background(), openRequest(), resp)
			if err != nil {
				return
			}
			if h, ok := handle.(interfaces_handle.FileHandle); ok {
				h.Close()
			}
		}()
	}

	go func() {
		defer wg.Done()
		<-start
		node.Close()
	}()

	close(start)
	wg.Wait()
}

// TestHandleId_Unique verifies that IDs allocated across concurrent NewHandle
// calls are all distinct.
func TestHandleId_Unique(t *testing.T) {
	t.Parallel()

	node, _ := newTestFileNode()

	const goroutines = 50
	ids := make(chan uint64, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			resp := &fuse.OpenResponse{}
			handle, err := node.Open(context.Background(), openRequest(), resp)
			if err != nil || handle == nil {
				return
			}
			if h, ok := handle.(interfaces_handle.FileHandle); ok {
				ids <- h.GetIdentifier()
			}
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[uint64]bool)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate handle ID: %d", id)
		}
		seen[id] = true
	}
}

// Compile-time check: Node satisfies fs.Node.
var _ fs.Node = (*Node)(nil)
