package registry

import (
	"os"
	"sync"
	"testing"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
)

// stubNode is a minimal AbstractNode that satisfies the full interface.
type stubNode struct {
	mu     sync.Mutex
	closed bool
}

func (n *stubNode) GetIdentifier() uint64                          { return 0 }
func (n *stubNode) GetRemoteIdentifier() uint64                    { return 0 }
func (n *stubNode) GetClient() interfaces_filesystem_client.Client { return nil }
func (n *stubNode) GetMode() os.FileMode                           { return 0 }
func (n *stubNode) IsClosed() bool                                 { n.mu.Lock(); defer n.mu.Unlock(); return n.closed }
func (n *stubNode) Close() error                                   { n.mu.Lock(); defer n.mu.Unlock(); n.closed = true; return nil }

var _ interfaces_node.AbstractNode = (*stubNode)(nil)

// TestAdd_Concurrent exercises concurrent Add calls. Before the sync.RWMutex
// fix, the race detector flags a fatal concurrent map/slice write here.
func TestAdd_Concurrent(t *testing.T) {
	t.Parallel()

	registry := New()

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			registry.Add(&stubNode{})
		}()
	}

	wg.Wait()
}

// TestCloseNodes_Concurrent calls CloseNodes while Add is still in flight.
// Before the RWMutex fix this could deadlock or corrupt the slice.
func TestCloseNodes_Concurrent(t *testing.T) {
	t.Parallel()

	registry := New()

	// Pre-populate so CloseNodes has work to do.
	for i := 0; i < 50; i++ {
		registry.Add(&stubNode{})
	}

	var wg sync.WaitGroup

	// Concurrent Adds.
	const addGoroutines = 50
	wg.Add(addGoroutines)
	for i := 0; i < addGoroutines; i++ {
		go func() {
			defer wg.Done()
			registry.Add(&stubNode{})
		}()
	}

	// CloseNodes races with the adds.
	wg.Add(1)
	go func() {
		defer wg.Done()
		registry.CloseNodes()
	}()

	wg.Wait()
}
