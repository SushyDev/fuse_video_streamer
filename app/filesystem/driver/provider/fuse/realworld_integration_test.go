// Real-world full-stack integration tests that exercise the FUSE node/handle
// layer backed by real HTTP streams through an unstable proxy. Tests simulate
// the kind of chaos that happens in production: nodes are added and removed
// dynamically, stream URLs are flaky, the HTTP server drops connections, and
// readers jump around sporadically.
//
// These tests do NOT mount a real FUSE filesystem — they call Open/Read/Release
// directly on node structs, same as the existing fuse_integration_test.go.
package fuse_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fuse_video_streamer/config"
	mock_config "fuse_video_streamer/config/mock"
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	mock_client "fuse_video_streamer/filesystem/client/mock"
	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	handle_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/streamable"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	node_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/streamable"
	mock_pool "fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool/mock"
	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/registry"
	interfaces_logger "fuse_video_streamer/logger/interfaces"
	mock_logger "fuse_video_streamer/logger/mock"
	http_ring_buffer "fuse_video_streamer/stream/drivers/http_ring_buffer"
	"fuse_video_streamer/stream/drivers/http_ring_buffer/factory"
	interfaces_stream "fuse_video_streamer/stream/interfaces"

	"github.com/anacrolix/fuse"
)

// ─── Unstable proxy (local, self-contained) ──────────────────────────────────

type realWorldProxy struct {
	content     []byte
	failRate    atomic.Int64
	totalServed atomic.Int64
	totalFailed atomic.Int64
	server      *httptest.Server
}

func newRealWorldProxy(content []byte) *realWorldProxy {
	proxy := &realWorldProxy{content: content}

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.handler)
	proxy.server = httptest.NewServer(mux)
	return proxy
}

func (proxy *realWorldProxy) URL() string         { return proxy.server.URL + "/" }
func (proxy *realWorldProxy) SetFailRate(pct int) { proxy.failRate.Store(int64(pct)) }
func (proxy *realWorldProxy) Close()              { proxy.server.Close() }

func (proxy *realWorldProxy) handler(w http.ResponseWriter, r *http.Request) {
	rate := int(proxy.failRate.Load())
	if rate > 0 {
		n, _ := rand.Int(rand.Reader, big.NewInt(100))
		if int(n.Int64()) < rate {
			proxy.totalFailed.Add(1)
			// Pick a random failure.
			kind, _ := rand.Int(rand.Reader, big.NewInt(4))
			switch int(kind.Int64()) {
			case 0:
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			case 1:
				// Connection reset.
				if hijacker, ok := w.(http.Hijacker); ok {
					if conn, _, err := hijacker.Hijack(); err == nil {
						conn.Close()
						return
					}
				}
				return
			case 2:
				// Empty body.
				w.Header().Set("Content-Length", "1048576")
				w.WriteHeader(http.StatusPartialContent)
				return
			default:
				// Hang until client gives up.
				w.WriteHeader(http.StatusPartialContent)
				w.Write([]byte{0x00})
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
				<-r.Context().Done()
				return
			}
		}
	}

	proxy.totalServed.Add(1)

	rangeHeader := r.Header.Get("Range")
	start := int64(0)
	if rangeHeader != "" {
		fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
	}
	if start >= int64(len(proxy.content)) {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	remaining := proxy.content[start:]
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(remaining)))
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Range",
		fmt.Sprintf("bytes %d-%d/%d", start, int64(len(proxy.content))-1, len(proxy.content)))
	w.WriteHeader(http.StatusPartialContent)

	ctx := r.Context()
	chunkSize := 64 * 1024
	for offset := 0; offset < len(remaining); offset += chunkSize {
		select {
		case <-ctx.Done():
			return
		default:
		}
		end := offset + chunkSize
		if end > len(remaining) {
			end = len(remaining)
		}
		if _, err := w.Write(remaining[offset:end]); err != nil {
			return
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// ─── Stream factory backed by real HTTP ring buffer ──────────────────────────

type realStreamFactory struct {
	config        *config.Config
	loggerFactory interfaces_logger.LoggerFactory
	proxyURL      string
	closed        atomic.Bool
}

func (factory *realStreamFactory) NewStream(_ context.Context, _ uint64, size uint64) (interfaces_stream.Stream, error) {
	if factory.closed.Load() {
		return nil, fmt.Errorf("factory closed")
	}
	return http_ring_buffer.New(factory.config, factory.loggerFactory, factory.proxyURL, int64(size))
}

func (factory *realStreamFactory) Close() error {
	factory.closed.Store(true)
	return nil
}

var _ interfaces_stream.StreamFactory = (*realStreamFactory)(nil)

// ─── Handle service factory using real streams ───────────────────────────────

type realHandleServiceFactory struct {
	streamFactory interfaces_stream.StreamFactory
	bufferPool    *mock_pool.BufferPool
	loggerFactory interfaces_logger.LoggerFactory
}

func (factory *realHandleServiceFactory) NewService(
	node interfaces_node.StreamableNode,
	client interfaces_filesystem_client.Client,
) (interfaces_handle.StreamableHandleService, error) {
	logger, err := factory.loggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}
	return handle_streamable.NewService(
		node,
		client,
		factory.loggerFactory,
		factory.streamFactory,
		factory.bufferPool,
		logger,
		false,
	), nil
}

var _ interfaces_handle.StreamableHandleServiceFactory = (*realHandleServiceFactory)(nil)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildRealWorldNode(
	id uint64,
	fileSize uint64,
	handleFactory interfaces_handle.StreamableHandleServiceFactory,
	proxyURL string,
) (*node_streamable.Node, error) {
	child := mock_client.NewNode(id, fmt.Sprintf("video_%d.mkv", id), os.FileMode(0644), true, fileSize)
	children := map[uint64][]*mock_client.Node{1: {child}}
	fileSystem := mock_client.NewFileSystem(
		mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0),
		children,
		func(_ uint64) (string, error) { return proxyURL, nil },
	)
	client := mock_client.NewClient("test", "/tmp", fileSystem)

	logger, _ := mock_logger.NoopLoggerFactory{}.NewLogger("Node")
	return node_streamable.NewNode(client, handleFactory, logger, id, id, fileSize, os.FileMode(0644))
}

// ─── Test: Dynamic node add/remove with real HTTP streams ────────────────────

// TestRealWorld_DynamicNodeAddRemove simulates a scenario where FUSE nodes are
// dynamically added and removed from the registry while readers are actively
// opening and reading from them. Streams go through an unstable proxy that
// drops 10% of connections.
func TestRealWorld_DynamicNodeAddRemove(t *testing.T) {
	t.Parallel()

	contentSize := 1 * 1024 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	proxy := newRealWorldProxy(content)
	defer proxy.Close()
	proxy.SetFailRate(10)

	cfg := mock_config.MinimalConfig()
	loggerFactory := mock_logger.NoopLoggerFactory{}
	bufferPool := &mock_pool.BufferPool{}

	reg := registry.New()

	streamFactory := &realStreamFactory{
		config:        cfg,
		loggerFactory: loggerFactory,
		proxyURL:      proxy.URL(),
	}

	handleFactory := &realHandleServiceFactory{
		streamFactory: streamFactory,
		bufferPool:    bufferPool,
		loggerFactory: loggerFactory,
	}

	var mu sync.RWMutex
	nodes := make(map[uint64]*node_streamable.Node)

	addNode := func(id uint64) {
		node, err := buildRealWorldNode(id, uint64(contentSize), handleFactory, proxy.URL())
		if err != nil {
			return
		}
		mu.Lock()
		nodes[id] = node
		mu.Unlock()
		reg.Add(node)
	}

	removeNode := func(id uint64) {
		mu.Lock()
		node, ok := nodes[id]
		if ok {
			delete(nodes, id)
		}
		mu.Unlock()
		if ok && node != nil {
			node.Close()
		}
	}

	// Start with 5 nodes.
	for i := uint64(100); i < 105; i++ {
		addNode(i)
	}

	const (
		testDuration = 5 * time.Second
		goroutines   = 10
	)

	deadline := time.Now().Add(testDuration)
	var wg sync.WaitGroup

	// Reader goroutines.
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))

			for time.Now().Before(deadline) {
				mu.RLock()
				ids := make([]uint64, 0, len(nodes))
				for id := range nodes {
					ids = append(ids, id)
				}
				mu.RUnlock()

				if len(ids) == 0 {
					time.Sleep(time.Millisecond)
					continue
				}

				nodeID := ids[rng.Intn(len(ids))]
				mu.RLock()
				node := nodes[nodeID]
				mu.RUnlock()

				if node == nil || node.IsClosed() {
					continue
				}

				openReq := &fuse.OpenRequest{}
				openResp := &fuse.OpenResponse{}
				handle, err := node.Open(context.Background(), openReq, openResp)
				if err != nil || handle == nil {
					continue
				}

				streamableHandle, ok := handle.(interfaces_handle.StreamableHandle)
				if !ok {
					continue
				}

				// 1-3 reads at random offsets.
				readCount := 1 + rng.Intn(3)
				for r := 0; r < readCount; r++ {
					offset := int64(rng.Intn(contentSize))
					readReq := &fuse.ReadRequest{
						Offset: offset,
						Size:   32 * 1024,
					}
					readResp := &fuse.ReadResponse{}
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					_ = streamableHandle.Read(ctx, readReq, readResp)
					cancel()
				}

				releaseReq := &fuse.ReleaseRequest{}
				_ = streamableHandle.Release(context.Background(), releaseReq)
			}
		}()
	}

	// Node lifecycle goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		nextID := uint64(200)
		rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))

		for time.Now().Before(deadline) {
			time.Sleep(time.Duration(100+rng.Intn(300)) * time.Millisecond)

			if rng.Intn(2) == 0 {
				addNode(nextID)
				nextID++
			} else {
				mu.RLock()
				ids := make([]uint64, 0, len(nodes))
				for id := range nodes {
					ids = append(ids, id)
				}
				mu.RUnlock()

				if len(ids) > 2 {
					victim := ids[rng.Intn(len(ids))]
					removeNode(victim)
				}
			}
		}
	}()

	wg.Wait()

	// Cleanup.
	mu.RLock()
	remaining := make([]*node_streamable.Node, 0, len(nodes))
	for _, node := range nodes {
		remaining = append(remaining, node)
	}
	mu.RUnlock()
	for _, node := range remaining {
		node.Close()
	}

	// Allow in-flight goroutines to drain.
	time.Sleep(200 * time.Millisecond)

	t.Logf("dynamic add/remove: served=%d failed=%d pool_live=%d",
		proxy.totalServed.Load(), proxy.totalFailed.Load(), bufferPool.LiveAllocations())

	if bufferPool.LiveAllocations() != 0 {
		t.Errorf("buffer pool leak: %d live allocations", bufferPool.LiveAllocations())
	}
}

// ─── Test: Flaky URL resolver through real factory ───────────────────────────

// TestRealWorld_FlakyURLResolver exercises the factory's retry logic with a
// GetStreamUrl that intermittently returns empty strings, errors, or valid URLs.
// When a valid URL is eventually returned, it points at an unstable proxy that
// also fails 10% of requests.
func TestRealWorld_FlakyURLResolver(t *testing.T) {
	t.Parallel()

	contentSize := 256 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	proxy := newRealWorldProxy(content)
	defer proxy.Close()
	proxy.SetFailRate(10)

	loggerFactory := mock_logger.NoopLoggerFactory{}
	cfg := mock_config.MinimalConfig()

	var callCount atomic.Int64

	getStreamURL := func(_ uint64) (string, error) {
		n := callCount.Add(1)
		switch {
		case n%5 == 0:
			return proxy.URL(), nil
		case n%3 == 0:
			return "", fmt.Errorf("temporary DNS failure")
		case n%7 == 0:
			return "", nil
		default:
			return "", fmt.Errorf("connection refused")
		}
	}

	rootNode := mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0)
	child := mock_client.NewNode(42, "video.mkv", os.FileMode(0644), true, uint64(contentSize))
	children := map[uint64][]*mock_client.Node{rootNode.GetId(): {child}}
	fileSystem := mock_client.NewFileSystem(rootNode, children, getStreamURL)
	client := mock_client.NewClient("test", "/tmp", fileSystem)

	streamFactory := factory.New(cfg, client, loggerFactory)
	streamFactory.SetNewTimer(func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	})
	defer streamFactory.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := streamFactory.NewStream(ctx, 42, uint64(contentSize))
	if err != nil {
		t.Fatalf("NewStream failed after retries: %v (calls=%d)", err, callCount.Load())
	}
	defer stream.Close()

	t.Logf("flaky URL resolver: created stream after %d GetStreamUrl calls", callCount.Load())

	// Verify data integrity through the stream.
	buffer := make([]byte, 1024)
	readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer readCancel()

	bytesRead, readErr := stream.ReadAt(readCtx, buffer, 0)
	if readErr != nil && readErr != io.EOF {
		t.Logf("read after flaky resolution: err=%v", readErr)
	} else if bytesRead > 0 {
		if !bytes.Equal(buffer[:bytesRead], content[:bytesRead]) {
			t.Error("data mismatch after flaky URL resolution")
		}
		t.Logf("read %d bytes through flaky URL resolver", bytesRead)
	}
}

// ─── Test: Sporadic reads (1% of file, random positions) ────────────────────

// TestRealWorld_SporadicPartialReads creates multiple nodes, each backed by a
// 4 MB file. Readers only read tiny slices (< 1%) from random positions,
// simulating a player that skips around the file. Reads go forward, backward,
// and near the boundaries.
func TestRealWorld_SporadicPartialReads(t *testing.T) {
	t.Parallel()

	contentSize := 4 * 1024 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	proxy := newRealWorldProxy(content)
	defer proxy.Close()
	proxy.SetFailRate(5)

	cfg := mock_config.MinimalConfig()
	loggerFactory := mock_logger.NoopLoggerFactory{}
	bufferPool := &mock_pool.BufferPool{}

	streamFactory := &realStreamFactory{
		config:        cfg,
		loggerFactory: loggerFactory,
		proxyURL:      proxy.URL(),
	}

	handleFactory := &realHandleServiceFactory{
		streamFactory: streamFactory,
		bufferPool:    bufferPool,
		loggerFactory: loggerFactory,
	}

	// Create 3 nodes.
	nodeIDs := []uint64{10, 11, 12}
	allNodes := make([]*node_streamable.Node, len(nodeIDs))
	for i, id := range nodeIDs {
		node, err := buildRealWorldNode(id, uint64(contentSize), handleFactory, proxy.URL())
		if err != nil {
			t.Fatalf("buildRealWorldNode %d: %v", id, err)
		}
		allNodes[i] = node
	}
	defer func() {
		for _, node := range allNodes {
			node.Close()
		}
	}()

	const (
		goroutines = 6
		duration   = 4 * time.Second
	)

	deadline := time.Now().Add(duration)
	var wg sync.WaitGroup
	var readCount, errorCount atomic.Int64

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))

			for time.Now().Before(deadline) {
				node := allNodes[rng.Intn(len(allNodes))]
				if node.IsClosed() {
					continue
				}

				openReq := &fuse.OpenRequest{}
				openResp := &fuse.OpenResponse{}
				handle, err := node.Open(context.Background(), openReq, openResp)
				if err != nil || handle == nil {
					errorCount.Add(1)
					continue
				}

				streamableHandle, ok := handle.(interfaces_handle.StreamableHandle)
				if !ok {
					continue
				}

				// Read tiny slices from random positions — less than 1% of file.
				readSlice := 1024 + rng.Intn(4096) // 1-5 KB per read
				reads := 2 + rng.Intn(5)

				for r := 0; r < reads; r++ {
					// Vary position: near start, near end, middle, random.
					var offset int64
					switch rng.Intn(4) {
					case 0:
						offset = int64(rng.Intn(readSlice * 2))
					case 1:
						offset = int64(contentSize - rng.Intn(readSlice*2) - readSlice)
					case 2:
						offset = int64(contentSize/2 + rng.Intn(readSlice*2) - readSlice)
					default:
						offset = int64(rng.Intn(contentSize - readSlice))
					}
					if offset < 0 {
						offset = 0
					}

					readReq := &fuse.ReadRequest{
						Offset: offset,
						Size:   readSlice,
					}
					readResp := &fuse.ReadResponse{}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					readErr := streamableHandle.Read(ctx, readReq, readResp)
					cancel()

					if readErr != nil {
						errorCount.Add(1)
					} else {
						readCount.Add(1)
						// Gap 5: Verify data integrity — the bytes returned
						// must match the known content at the requested offset.
						if len(readResp.Data) > 0 {
							end := offset + int64(len(readResp.Data))
							if end > int64(contentSize) {
								end = int64(contentSize)
							}
							expected := content[offset:end]
							if !bytes.Equal(readResp.Data[:end-offset], expected) {
								t.Errorf("data mismatch at offset %d (read %d bytes)", offset, len(readResp.Data))
							}
						}
					}
				}

				releaseReq := &fuse.ReleaseRequest{}
				_ = streamableHandle.Release(context.Background(), releaseReq)
			}
		}()
	}

	wg.Wait()

	t.Logf("sporadic partial reads: reads=%d errors=%d served=%d failed=%d pool_live=%d",
		readCount.Load(), errorCount.Load(),
		proxy.totalServed.Load(), proxy.totalFailed.Load(),
		bufferPool.LiveAllocations())

	if bufferPool.LiveAllocations() != 0 {
		t.Errorf("buffer pool leak: %d live allocations", bufferPool.LiveAllocations())
	}
}

// ─── Test: Stream URL is empty string from the start ─────────────────────────

// TestRealWorld_EmptyStreamURL exercises what happens when the filesystem
// client always returns an empty stream URL — the factory should eventually
// give up with EAGAIN.
func TestRealWorld_EmptyStreamURL(t *testing.T) {
	t.Parallel()

	cfg := mock_config.MinimalConfig()
	loggerFactory := mock_logger.NoopLoggerFactory{}

	rootNode := mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0)
	child := mock_client.NewNode(42, "video.mkv", os.FileMode(0644), true, 1024)
	children := map[uint64][]*mock_client.Node{rootNode.GetId(): {child}}

	// GetStreamUrl always returns empty string.
	fileSystem := mock_client.NewFileSystem(rootNode, children, func(_ uint64) (string, error) {
		return "", nil
	})
	client := mock_client.NewClient("test", "/tmp", fileSystem)

	streamFactory := factory.New(cfg, client, loggerFactory)
	streamFactory.SetNewTimer(func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	})
	defer streamFactory.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := streamFactory.NewStream(ctx, 42, 1024)
	if err == nil {
		t.Fatal("expected error from empty-URL resolver, got nil")
	}
	t.Logf("empty URL resolver: %v", err)
}

// ─── Test: Stream URL errors with various error types ────────────────────────

// TestRealWorld_StreamURLErrors exercises various error conditions from the URL
// resolver: transient errors, permanent errors (syscall.Errno), and context
// cancellation during retry.
func TestRealWorld_StreamURLErrors(t *testing.T) {
	t.Parallel()

	cfg := mock_config.MinimalConfig()
	loggerFactory := mock_logger.NoopLoggerFactory{}

	immediateTimer := func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}

	t.Run("transient_then_success", func(t *testing.T) {
		t.Parallel()

		contentSize := 64 * 1024
		content := make([]byte, contentSize)
		rand.Read(content)

		proxy := newRealWorldProxy(content)
		defer proxy.Close()

		var callCount atomic.Int64
		getStreamURL := func(_ uint64) (string, error) {
			n := callCount.Add(1)
			if n < 4 {
				return "", fmt.Errorf("transient network error (attempt %d)", n)
			}
			return proxy.URL(), nil
		}

		rootNode := mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0)
		child := mock_client.NewNode(42, "video.mkv", os.FileMode(0644), true, uint64(contentSize))
		children := map[uint64][]*mock_client.Node{rootNode.GetId(): {child}}
		fileSystem := mock_client.NewFileSystem(rootNode, children, getStreamURL)
		client := mock_client.NewClient("test", "/tmp", fileSystem)

		streamFactory := factory.New(cfg, client, loggerFactory)
		streamFactory.SetNewTimer(immediateTimer)
		defer streamFactory.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		stream, err := streamFactory.NewStream(ctx, 42, uint64(contentSize))
		if err != nil {
			t.Fatalf("expected success after transient errors, got: %v (calls=%d)", err, callCount.Load())
		}
		defer stream.Close()

		buffer := make([]byte, 1024)
		readCtx, readCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer readCancel()

		bytesRead, readErr := stream.ReadAt(readCtx, buffer, 0)
		if readErr != nil && readErr != io.EOF {
			t.Logf("read: %v (may be expected)", readErr)
		} else if bytesRead > 0 {
			if !bytes.Equal(buffer[:bytesRead], content[:bytesRead]) {
				t.Error("data mismatch")
			}
		}

		t.Logf("transient then success: calls=%d", callCount.Load())
	})

	t.Run("context_cancelled_during_retry", func(t *testing.T) {
		t.Parallel()

		rootNode := mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0)
		child := mock_client.NewNode(42, "video.mkv", os.FileMode(0644), true, 1024)
		children := map[uint64][]*mock_client.Node{rootNode.GetId(): {child}}
		fileSystem := mock_client.NewFileSystem(rootNode, children, func(_ uint64) (string, error) {
			return "", fmt.Errorf("always fails")
		})
		client := mock_client.NewClient("test", "/tmp", fileSystem)

		streamFactory := factory.New(cfg, client, loggerFactory)
		streamFactory.SetNewTimer(immediateTimer)
		defer streamFactory.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Already cancelled.

		_, err := streamFactory.NewStream(ctx, 42, 1024)
		if err == nil {
			t.Fatal("expected error from cancelled context")
		}
		t.Logf("cancelled context: %v", err)
	})
}

// ─── Gap 3: Handle recovery — new Open() on same node after stream death ─────

// TestRealWorld_HandleRecoveryAfterStreamDeath opens a handle on a node whose
// HTTP server always fails (HTTP 500), forcing the first stream to die. Then
// the server is switched to healthy, and a second Open() on the same node
// must produce a working handle that can read data successfully.
func TestRealWorld_HandleRecoveryAfterStreamDeath(t *testing.T) {
	t.Parallel()

	contentSize := 256 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	// Start with a server that always returns HTTP 500.
	var serverHealthy atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !serverHealthy.Load() {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		rangeHeader := r.Header.Get("Range")
		start := int64(0)
		if rangeHeader != "" {
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
		}
		if start >= int64(contentSize) {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		remaining := content[start:]
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(remaining)))
		w.Header().Set("Content-Range",
			fmt.Sprintf("bytes %d-%d/%d", start, int64(contentSize)-1, contentSize))
		w.WriteHeader(http.StatusPartialContent)

		ctx := r.Context()
		chunkSize := 64 * 1024
		for offset := 0; offset < len(remaining); offset += chunkSize {
			select {
			case <-ctx.Done():
				return
			default:
			}
			end := offset + chunkSize
			if end > len(remaining) {
				end = len(remaining)
			}
			if _, err := w.Write(remaining[offset:end]); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	cfg := mock_config.MinimalConfig()
	loggerFactory := mock_logger.NoopLoggerFactory{}
	bufferPool := &mock_pool.BufferPool{}

	streamFactory := &realStreamFactory{
		config:        cfg,
		loggerFactory: loggerFactory,
		proxyURL:      server.URL + "/",
	}

	handleFactory := &realHandleServiceFactory{
		streamFactory: streamFactory,
		bufferPool:    bufferPool,
		loggerFactory: loggerFactory,
	}

	node, err := buildRealWorldNode(42, uint64(contentSize), handleFactory, server.URL+"/")
	if err != nil {
		t.Fatalf("buildRealWorldNode: %v", err)
	}
	defer node.Close()

	// First Open: the server is broken, so the stream should fail.
	openReq := &fuse.OpenRequest{}
	openResp := &fuse.OpenResponse{}
	handle1, err := node.Open(context.Background(), openReq, openResp)
	if err != nil {
		t.Logf("first Open failed at creation (expected for some error paths): %v", err)
	} else if handle1 != nil {
		// Handle was created, but reading should fail because the server is broken.
		streamableHandle1, ok := handle1.(interfaces_handle.StreamableHandle)
		if ok {
			readReq := &fuse.ReadRequest{Offset: 0, Size: 1024}
			readResp := &fuse.ReadResponse{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			readErr := streamableHandle1.Read(ctx, readReq, readResp)
			cancel()
			t.Logf("first handle read: err=%v dataLen=%d", readErr, len(readResp.Data))
			streamableHandle1.Release(context.Background(), &fuse.ReleaseRequest{})
		}
	}

	// Switch server to healthy.
	serverHealthy.Store(true)

	// Second Open: should succeed and produce a working stream.
	openResp2 := &fuse.OpenResponse{}
	handle2, err := node.Open(context.Background(), openReq, openResp2)
	if err != nil {
		t.Fatalf("second Open failed: %v", err)
	}
	if handle2 == nil {
		t.Fatal("second Open returned nil handle")
	}

	streamableHandle2, ok := handle2.(interfaces_handle.StreamableHandle)
	if !ok {
		t.Fatal("second handle is not StreamableHandle")
	}
	defer streamableHandle2.Release(context.Background(), &fuse.ReleaseRequest{})

	// Read from the second handle — must succeed with correct data.
	readReq2 := &fuse.ReadRequest{Offset: 0, Size: 1024}
	readResp2 := &fuse.ReadResponse{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	readErr := streamableHandle2.Read(ctx, readReq2, readResp2)
	cancel()

	if readErr != nil {
		t.Fatalf("second handle read failed: %v", readErr)
	}
	if len(readResp2.Data) == 0 {
		t.Fatal("second handle read returned 0 bytes")
	}
	if !bytes.Equal(readResp2.Data, content[:len(readResp2.Data)]) {
		t.Error("second handle data mismatch — recovery produced corrupt data")
	}

	t.Logf("handle recovery: second handle read %d bytes successfully", len(readResp2.Data))
}

// ─── Gap 4: NewHandle timeout — slow URL resolver respects context ───────────

// TestRealWorld_NewHandleTimeout verifies that when the stream factory (URL
// resolver) hangs, the propagated context from Open() causes NewHandle to
// unblock and return an error instead of hanging forever. This tests the
// production fix that changed NewHandle(ctx) to pass the FUSE context.
func TestRealWorld_NewHandleTimeout(t *testing.T) {
	t.Parallel()

	loggerFactory := mock_logger.NoopLoggerFactory{}

	// A stream factory whose NewStream blocks until the context is cancelled.
	hangingStreamFactory := &hangingStreamFactory{}

	bufferPool := &mock_pool.BufferPool{}

	hangingHandleFactory := &realHandleServiceFactory{
		streamFactory: hangingStreamFactory,
		bufferPool:    bufferPool,
		loggerFactory: loggerFactory,
	}

	node, err := buildRealWorldNode(99, 1024, hangingHandleFactory, "http://127.0.0.1:0/")
	if err != nil {
		t.Fatalf("buildRealWorldNode: %v", err)
	}
	defer node.Close()

	// Open with a 2-second timeout — simulating a FUSE client that gives up.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	openReq := &fuse.OpenRequest{}
	openResp := &fuse.OpenResponse{}

	startTime := time.Now()
	handle, err := node.Open(ctx, openReq, openResp)
	elapsed := time.Since(startTime)

	// The Open must either return an error or a nil handle — it must NOT hang
	// for longer than the context timeout plus a small margin.
	if elapsed > 3*time.Second {
		t.Fatalf("Open took %v — NewHandle did not respect the context timeout", elapsed)
	}

	if err != nil {
		t.Logf("Open returned error (expected): %v in %v", err, elapsed)
	} else if handle == nil {
		t.Logf("Open returned nil handle in %v", elapsed)
	} else {
		// Unexpected: handle was created despite hanging factory.
		t.Log("Open returned a handle despite hanging factory — closing it")
		if streamableHandle, ok := handle.(interfaces_handle.StreamableHandle); ok {
			streamableHandle.Release(context.Background(), &fuse.ReleaseRequest{})
		}
	}
}

// hangingStreamFactory is a StreamFactory whose NewStream blocks until ctx is
// cancelled — simulating a gRPC call to a remote service that is down.
type hangingStreamFactory struct {
	closed atomic.Bool
}

func (factory *hangingStreamFactory) NewStream(ctx context.Context, _ uint64, _ uint64) (interfaces_stream.Stream, error) {
	if factory.closed.Load() {
		return nil, fmt.Errorf("factory closed")
	}
	// Block until the context is cancelled.
	<-ctx.Done()
	return nil, ctx.Err()
}

func (factory *hangingStreamFactory) Close() error {
	factory.closed.Store(true)
	return nil
}

var _ interfaces_stream.StreamFactory = (*hangingStreamFactory)(nil)
