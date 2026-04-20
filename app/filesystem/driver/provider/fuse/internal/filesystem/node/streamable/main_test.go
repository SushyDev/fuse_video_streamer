package streamable

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

// stubStreamableHandle is a minimal StreamableHandle.
type stubStreamableHandle struct {
	id uint64
}

func (h *stubStreamableHandle) GetIdentifier() uint64 { return h.id }
func (h *stubStreamableHandle) Close() error          { return nil }
func (h *stubStreamableHandle) IsClosed() bool        { return false }
func (h *stubStreamableHandle) Read(_ context.Context, _ *fuse.ReadRequest, _ *fuse.ReadResponse) error {
	return nil
}
func (h *stubStreamableHandle) Release(_ context.Context, _ *fuse.ReleaseRequest) error {
	return nil
}

var _ interfaces_handle.StreamableHandle = (*stubStreamableHandle)(nil)

// stubStreamableHandleService creates stubStreamableHandle instances.
type stubStreamableHandleService struct {
	mu      sync.Mutex
	counter uint64
	closed  bool
}

func (service *stubStreamableHandleService) NewHandle(_ context.Context) (interfaces_handle.StreamableHandle, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.closed {
		return nil, nil
	}
	service.counter++
	return &stubStreamableHandle{id: service.counter}, nil
}

func (service *stubStreamableHandleService) Close() error {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.closed = true
	return nil
}

func (service *stubStreamableHandleService) IsClosed() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.closed
}

var _ interfaces_handle.StreamableHandleService = (*stubStreamableHandleService)(nil)

// stubStreamableHandleServiceFactory creates stubStreamableHandleService instances.
type stubStreamableHandleServiceFactory struct{}

func (stubStreamableHandleServiceFactory) NewService(
	_ interfaces_node.StreamableNode,
	_ interfaces_filesystem_client.Client,
) (interfaces_handle.StreamableHandleService, error) {
	return &stubStreamableHandleService{}, nil
}

var _ interfaces_handle.StreamableHandleServiceFactory = stubStreamableHandleServiceFactory{}

// newTestStreamableNode creates a Node wired with stub dependencies.
func newTestStreamableNode() (*Node, error) {
	return NewNode(
		nil, // client — not exercised in these tests
		stubStreamableHandleServiceFactory{},
		mock_logger.NoopLogger{},
		1,    // identifier
		100,  // remoteIdentifier
		1024, // size
		0644,
	)
}

func openRequest() *fuse.OpenRequest {
	return &fuse.OpenRequest{}
}

// TestOpen_Concurrent calls Open from N goroutines simultaneously.
func TestOpen_Concurrent(t *testing.T) {
	t.Parallel()

	node, err := newTestStreamableNode()
	if err != nil {
		t.Fatalf("newTestStreamableNode: %v", err)
	}

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
			if h, ok := handle.(interfaces_handle.StreamableHandle); ok {
				h.Close()
			}
		}()
	}

	wg.Wait()
}

// TestClose_WhileOpenInFlight calls Close while Open goroutines are running.
func TestClose_WhileOpenInFlight(t *testing.T) {
	t.Parallel()

	node, err := newTestStreamableNode()
	if err != nil {
		t.Fatalf("newTestStreamableNode: %v", err)
	}

	const openGoroutines = 50
	var wg sync.WaitGroup
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
			if h, ok := handle.(interfaces_handle.StreamableHandle); ok {
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

// TestHandleId_Unique verifies that IDs across concurrent Open calls are all distinct.
func TestHandleId_Unique(t *testing.T) {
	t.Parallel()

	node, err := newTestStreamableNode()
	if err != nil {
		t.Fatalf("newTestStreamableNode: %v", err)
	}

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
			if h, ok := handle.(interfaces_handle.StreamableHandle); ok {
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
