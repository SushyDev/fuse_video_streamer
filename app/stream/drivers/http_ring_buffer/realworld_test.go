// Package http_ring_buffer_test contains real-world stress tests that exercise
// the full Stream stack (ring buffer, connection, transfer) against a
// combination of real Netflix demo content and local HTTP test servers that
// inject instability (empty URLs, HTTP 500s, connection drops, slow responses).
//
// Tests that contact the Netflix S3 bucket are gated behind -short (skipped
// when testing.Short() is true). Local-only tests use an embedded unstable
// proxy to simulate the same failures.
//
// All tests use t.Parallel() and the race detector.
package http_ring_buffer_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mock_config "fuse_video_streamer/config/mock"
	interfaces_logger "fuse_video_streamer/logger/interfaces"
	mock_logger "fuse_video_streamer/logger/mock"
	http_ring_buffer "fuse_video_streamer/stream/drivers/http_ring_buffer"
)

// ─── Netflix demo content ────────────────────────────────────────────────────

const (
	netflixURL  = "http://download.opencontent.netflix.com.s3.amazonaws.com/SolLevante/hdr10/SolLevante_HDR10_r2020_ST2084_UHD_24fps_1000nit.mov"
	netflixSize = int64(37_392_798_951)
)

// ─── Logger stubs ────────────────────────────────────────────────────────────

type testLogger struct{ t *testing.T }

func (l testLogger) Info(message string)             { l.t.Log("[INFO]", message) }
func (l testLogger) Warn(message string)             { l.t.Log("[WARN]", message) }
func (l testLogger) Error(message string, err error) { l.t.Log("[ERROR]", message, err) }
func (l testLogger) Fatal(message string, err error) { l.t.Log("[FATAL]", message, err) }
func (l testLogger) Debug(message string)            { l.t.Log("[DEBUG]", message) }

type testLoggerFactory struct{ t *testing.T }

func (f testLoggerFactory) NewLogger(_ string) (interfaces_logger.Logger, error) {
	return testLogger{t: f.t}, nil
}

var _ interfaces_logger.LoggerFactory = testLoggerFactory{}

// ─── Unstable proxy ──────────────────────────────────────────────────────────
//
// unstableProxy wraps content (in-memory or upstream URL) and injects
// failures randomly:
//   - HTTP 500 / 503
//   - connection reset (hijack + close)
//   - empty body after headers
//   - hung response (timeout)
//   - normal pass-through

type failureKind int

const (
	failNone failureKind = iota
	failHTTP500
	failHTTP503
	failConnReset
	failEmptyBody
	failTimeout
)

type unstableProxy struct {
	upstream    string
	failRate    atomic.Int64
	totalServed atomic.Int64
	totalFailed atomic.Int64
	server      *httptest.Server
	content     []byte // if non-nil, serve from memory
}

func newUnstableProxy(upstream string, content []byte) *unstableProxy {
	proxy := &unstableProxy{
		upstream: upstream,
		content:  content,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", proxy.handler)
	proxy.server = httptest.NewServer(mux)
	return proxy
}

func (proxy *unstableProxy) URL() string         { return proxy.server.URL + "/" }
func (proxy *unstableProxy) SetFailRate(pct int) { proxy.failRate.Store(int64(pct)) }
func (proxy *unstableProxy) Close()              { proxy.server.Close() }

func (proxy *unstableProxy) pickFailure() failureKind {
	rate := int(proxy.failRate.Load())
	if rate <= 0 {
		return failNone
	}
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		return failNone
	}
	if int(n.Int64()) >= rate {
		return failNone
	}
	kindN, _ := rand.Int(rand.Reader, big.NewInt(5))
	switch int(kindN.Int64()) {
	case 0:
		return failHTTP500
	case 1:
		return failHTTP503
	case 2:
		return failConnReset
	case 3:
		return failEmptyBody
	default:
		return failTimeout
	}
}

func (proxy *unstableProxy) handler(w http.ResponseWriter, r *http.Request) {
	kind := proxy.pickFailure()

	switch kind {
	case failHTTP500:
		proxy.totalFailed.Add(1)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	case failHTTP503:
		proxy.totalFailed.Add(1)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	case failConnReset:
		proxy.totalFailed.Add(1)
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				conn.Close()
				return
			}
		}
		return
	case failEmptyBody:
		proxy.totalFailed.Add(1)
		w.Header().Set("Content-Length", "1048576")
		w.WriteHeader(http.StatusPartialContent)
		return
	case failTimeout:
		proxy.totalFailed.Add(1)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte{0x00, 0x01, 0x02})
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
		return
	}

	proxy.totalServed.Add(1)

	if proxy.content != nil {
		proxy.serveFromMemory(w, r)
		return
	}
	proxy.serveFromUpstream(w, r)
}

func (proxy *unstableProxy) serveFromMemory(w http.ResponseWriter, r *http.Request) {
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
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, int64(len(proxy.content))-1, len(proxy.content)))
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

func (proxy *unstableProxy) serveFromUpstream(w http.ResponseWriter, r *http.Request) {
	upstreamReq, err := http.NewRequestWithContext(r.Context(), "GET", proxy.upstream, nil)
	if err != nil {
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		upstreamReq.Header.Set("Range", rangeHeader)
	}

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newStream(t *testing.T, url string, size int64) *http_ring_buffer.Stream {
	t.Helper()
	cfg := mock_config.MinimalConfig()
	stream, err := http_ring_buffer.New(cfg, mock_logger.NoopLoggerFactory{}, url, size)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return stream
}

func newStreamWithLogger(t *testing.T, url string, size int64) *http_ring_buffer.Stream {
	t.Helper()
	cfg := mock_config.MinimalConfig()
	stream, err := http_ring_buffer.New(cfg, testLoggerFactory{t: t}, url, size)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return stream
}

// ─── Test: Real Netflix content, partial reads ───────────────────────────────

// TestRealWorld_Netflix_PartialRead reads a tiny fraction of the 37 GB Netflix
// file from a few random positions in the first 1%, verifying that range
// requests work and the stream handles seeks into a huge file.
func TestRealWorld_Netflix_PartialRead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-network test in short mode")
	}
	t.Parallel()

	stream := newStreamWithLogger(t, netflixURL, netflixSize)
	defer func() { stream.Close() }()

	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	readSize := 64 * 1024

	for i := 0; i < 5; i++ {
		maxOffset := netflixSize / 100
		offset := rng.Int63n(maxOffset)

		buffer := make([]byte, readSize)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		bytesRead, readErr := stream.ReadAt(ctx, buffer, offset)
		cancel()

		if readErr != nil && readErr != io.EOF {
			t.Logf("read %d at offset %d: err=%v — recreating stream", i, offset, readErr)
			stream.Close()
			stream = newStreamWithLogger(t, netflixURL, netflixSize)
			continue
		}

		if bytesRead == 0 {
			t.Logf("read %d at offset %d returned 0 bytes", i, offset)
			continue
		}

		t.Logf("read %d: offset=%d bytesRead=%d", i, offset, bytesRead)

		allZero := true
		for _, b := range buffer[:bytesRead] {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero && bytesRead > 16 {
			t.Errorf("read %d: all %d bytes are zero — likely corrupted", i, bytesRead)
		}
	}
}

// ─── Test: Sporadic random seeks forward and backward ────────────────────────

func TestRealWorld_Netflix_SporadicSeeks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-network test in short mode")
	}
	t.Parallel()

	stream := newStreamWithLogger(t, netflixURL, netflixSize)
	defer stream.Close()

	readSize := 32 * 1024
	buffer := make([]byte, readSize)

	seekPositions := []int64{
		0,
		1024,
		50 * 1024,
		10 * 1024 * 1024,
		1024,
		100 * 1024 * 1024,
		50 * 1024 * 1024,
		200 * 1024 * 1024,
	}

	for i, offset := range seekPositions {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		bytesRead, readErr := stream.ReadAt(ctx, buffer, offset)
		cancel()

		if readErr != nil && readErr != io.EOF {
			t.Logf("seek %d (offset %d): err=%v — recreating stream", i, offset, readErr)
			stream.Close()
			stream = newStreamWithLogger(t, netflixURL, netflixSize)
			continue
		}
		t.Logf("seek %d: offset=%d bytesRead=%d", i, offset, bytesRead)
	}
}

// ─── Test: Empty URL ─────────────────────────────────────────────────────────

func TestRealWorld_EmptyURL(t *testing.T) {
	t.Parallel()

	stream := newStream(t, "", 1024)
	defer stream.Close()

	buffer := make([]byte, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, readErr := stream.ReadAt(ctx, buffer, 0)
	if readErr == nil {
		t.Fatal("expected error from empty-URL stream, got nil")
	}
	t.Logf("empty URL: %v", readErr)
}

// ─── Test: Non-existent URL ──────────────────────────────────────────────────

func TestRealWorld_NonExistentURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
	}{
		{"unresolvable_host", "http://this-host-does-not-exist.invalid/video.mkv"},
		{"refused_connection", "http://127.0.0.1:1/video.mkv"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stream := newStream(t, tc.url, 1024)
			defer stream.Close()

			buffer := make([]byte, 64)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, readErr := stream.ReadAt(ctx, buffer, 0)
			if readErr == nil {
				t.Fatal("expected error, got nil")
			}
			t.Logf("%s: %v", tc.name, readErr)
		})
	}
}

// ─── Test: HTTP errors (non-200 range) ───────────────────────────────────────

func TestRealWorld_HTTPErrors(t *testing.T) {
	t.Parallel()

	codes := []int{
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
		http.StatusBadGateway,
	}

	for _, code := range codes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, fmt.Sprintf("error %d", code), code)
			}))
			defer server.Close()

			stream := newStream(t, server.URL+"/video.mkv", 1024*1024)
			defer stream.Close()

			buffer := make([]byte, 64)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, readErr := stream.ReadAt(ctx, buffer, 0)
			if readErr == nil {
				t.Fatalf("expected error for HTTP %d, got nil", code)
			}
			t.Logf("HTTP %d: %v", code, readErr)
		})
	}
}

// ─── Test: Connection drop mid-stream ────────────────────────────────────────

func TestRealWorld_ConnectionDrop(t *testing.T) {
	t.Parallel()

	bytesBeforeDrop := 8 * 1024

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				c.Read(buf)

				header := "HTTP/1.1 206 Partial Content\r\nContent-Length: 1048576\r\nContent-Type: application/octet-stream\r\n\r\n"
				c.Write([]byte(header))

				data := make([]byte, bytesBeforeDrop)
				for i := range data {
					data[i] = byte(i % 256)
				}
				c.Write(data)

				if tcpConn, ok := c.(*net.TCPConn); ok {
					tcpConn.SetLinger(0)
				}
			}(conn)
		}
	}()

	url := fmt.Sprintf("http://%s/video.mkv", listener.Addr().String())
	stream := newStreamWithLogger(t, url, 1024*1024)
	defer stream.Close()

	buffer := make([]byte, 128*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bytesRead, readErr := stream.ReadAt(ctx, buffer, 0)
	t.Logf("connection drop: bytesRead=%d err=%v", bytesRead, readErr)

	if bytesRead < 0 {
		t.Fatal("negative bytesRead")
	}
}

// ─── Test: Server hangs (timeout) ────────────────────────────────────────────

func TestRealWorld_ServerHangs(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.WriteHeader(http.StatusPartialContent)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	stream := newStream(t, server.URL+"/video.mkv", 1024*1024)
	defer stream.Close()

	buffer := make([]byte, 64)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, readErr := stream.ReadAt(ctx, buffer, 0)
	if readErr == nil {
		t.Fatal("expected timeout error, got nil")
	}
	t.Logf("hanging server: %v", readErr)
}

// ─── Test: Unstable proxy with local content ─────────────────────────────────

func TestRealWorld_UnstableProxy_LocalContent(t *testing.T) {
	t.Parallel()

	contentSize := 4 * 1024 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	t.Run("baseline_no_failures", func(t *testing.T) {
		t.Parallel()

		proxy := newUnstableProxy("", content)
		defer proxy.Close()
		proxy.SetFailRate(0)

		stream := newStream(t, proxy.URL(), int64(contentSize))
		defer stream.Close()

		readSize := 64 * 1024
		buffer := make([]byte, readSize)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		bytesRead, readErr := stream.ReadAt(ctx, buffer, 0)
		if readErr != nil && readErr != io.EOF {
			t.Fatalf("baseline read: %v", readErr)
		}
		if bytesRead == 0 {
			t.Fatal("baseline read returned 0 bytes")
		}
		if !bytes.Equal(buffer[:bytesRead], content[:bytesRead]) {
			t.Fatal("baseline data mismatch")
		}
	})

	t.Run("moderate_instability_30pct", func(t *testing.T) {
		t.Parallel()

		proxy := newUnstableProxy("", content)
		defer proxy.Close()
		proxy.SetFailRate(30)

		const goroutines = 5
		const readsPerGoroutine = 3
		const readTimeout = 15 * time.Second

		var wg sync.WaitGroup
		wg.Add(goroutines)

		var successCount, errorCount atomic.Int64

		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()

				stream, err := http_ring_buffer.New(
					mock_config.MinimalConfig(), mock_logger.NoopLoggerFactory{},
					proxy.URL(), int64(contentSize),
				)
				if err != nil {
					errorCount.Add(1)
					return
				}
				defer stream.Close()

				rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
				buffer := make([]byte, 32*1024)

				for i := 0; i < readsPerGoroutine; i++ {
					offset := int64(rng.Intn(contentSize - len(buffer)))
					ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
					bytesRead, readErr := stream.ReadAt(ctx, buffer, offset)
					cancel()

					if readErr != nil {
						errorCount.Add(1)
						stream.Close()
						stream, err = http_ring_buffer.New(
							mock_config.MinimalConfig(), mock_logger.NoopLoggerFactory{},
							proxy.URL(), int64(contentSize),
						)
						if err != nil {
							return
						}
						continue
					}

					if bytesRead > 0 {
						expected := content[offset : offset+int64(bytesRead)]
						if !bytes.Equal(buffer[:bytesRead], expected) {
							t.Errorf("data mismatch at offset %d", offset)
						}
						successCount.Add(1)
					}
				}
			}()
		}

		wg.Wait()

		t.Logf("30%% instability: successes=%d errors=%d served=%d failed=%d",
			successCount.Load(), errorCount.Load(),
			proxy.totalServed.Load(), proxy.totalFailed.Load())
	})

	t.Run("high_instability_60pct", func(t *testing.T) {
		t.Parallel()

		proxy := newUnstableProxy("", content)
		defer proxy.Close()
		proxy.SetFailRate(60)

		stream := newStream(t, proxy.URL(), int64(contentSize))
		defer stream.Close()

		buffer := make([]byte, 16*1024)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, readErr := stream.ReadAt(ctx, buffer, 0)
		t.Logf("60%% instability: err=%v", readErr)
	})
}

// ─── Test: Concurrent reads racing with stream close ─────────────────────────

func TestRealWorld_ConcurrentReads_RaceWithClose(t *testing.T) {
	t.Parallel()

	contentSize := 2 * 1024 * 1024
	content := make([]byte, contentSize)
	if _, err := rand.Read(content); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	proxy := newUnstableProxy("", content)
	defer proxy.Close()
	proxy.SetFailRate(5)

	const (
		testDuration = 4 * time.Second
		readers      = 8
	)

	deadline := time.Now().Add(testDuration)

	var currentStream atomic.Pointer[http_ring_buffer.Stream]
	stream := newStream(t, proxy.URL(), int64(contentSize))
	currentStream.Store(stream)

	var wg sync.WaitGroup

	wg.Add(readers)
	for g := 0; g < readers; g++ {
		go func() {
			defer wg.Done()
			rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
			buffer := make([]byte, 16*1024)

			for time.Now().Before(deadline) {
				s := currentStream.Load()
				if s == nil || s.IsClosed() {
					time.Sleep(time.Millisecond)
					continue
				}

				offset := int64(rng.Intn(contentSize - len(buffer)))
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_, _ = s.ReadAt(ctx, buffer, offset)
				cancel()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)

			old := currentStream.Load()
			if old != nil {
				old.Close()
			}

			newS, err := http_ring_buffer.New(
				mock_config.MinimalConfig(), mock_logger.NoopLoggerFactory{},
				proxy.URL(), int64(contentSize),
			)
			if err != nil {
				continue
			}
			currentStream.Store(newS)
		}
	}()

	wg.Wait()

	if s := currentStream.Load(); s != nil {
		s.Close()
	}
}

// ─── Test: Netflix through unstable proxy ────────────────────────────────────

func TestRealWorld_Netflix_UnstableProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-network test in short mode")
	}
	t.Parallel()

	proxy := newUnstableProxy(netflixURL, nil)
	defer proxy.Close()
	proxy.SetFailRate(20)

	stream := newStreamWithLogger(t, proxy.URL(), netflixSize)
	defer stream.Close()

	buffer := make([]byte, 64*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	bytesRead, readErr := stream.ReadAt(ctx, buffer, 0)
	if readErr != nil {
		t.Logf("Netflix unstable proxy: err=%v (served=%d failed=%d)",
			readErr, proxy.totalServed.Load(), proxy.totalFailed.Load())
	} else {
		t.Logf("Netflix unstable proxy: read %d bytes (served=%d failed=%d)",
			bytesRead, proxy.totalServed.Load(), proxy.totalFailed.Load())

		allZero := true
		for _, b := range buffer[:bytesRead] {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero && bytesRead > 16 {
			t.Error("all bytes are zero — likely corrupted")
		}
	}
}

// ─── Gap 1: Stream isolation — one stuck stream doesn't block another ────────

// TestStreamIsolation_StuckStreamDoesNotBlockOther creates two independent
// streams backed by separate HTTP servers. The first stream's server hangs
// indefinitely (simulating a stuck transfer). The second stream's server
// responds normally. We verify that reading from the second stream completes
// promptly even while the first stream is stuck.
func TestStreamIsolation_StuckStreamDoesNotBlockOther(t *testing.T) {
	t.Parallel()

	// Healthy server: 1 MB of deterministic content.
	contentSize := 1 * 1024 * 1024
	content := make([]byte, contentSize)
	for i := range content {
		content[i] = byte(i % 251) // deterministic, non-zero pattern
	}

	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer healthyServer.Close()

	// Stuck server: sends headers then hangs forever.
	stuckServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1048576")
		w.WriteHeader(http.StatusPartialContent)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		// Hang until the request context is cancelled (client gives up).
		<-r.Context().Done()
	}))
	defer stuckServer.Close()

	// Create the stuck stream first. Its ReadAt will block indefinitely.
	stuckStream := newStream(t, stuckServer.URL+"/video.mkv", int64(contentSize))
	defer stuckStream.Close()

	// Launch a goroutine that attempts to read from the stuck stream.
	// It should block until its context expires.
	stuckDone := make(chan struct{})
	go func() {
		defer close(stuckDone)
		buffer := make([]byte, 64*1024)
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		_, _ = stuckStream.ReadAt(ctx, buffer, 0)
	}()

	// Now read from the healthy stream. This must complete quickly — within
	// 3 seconds — even though the stuck stream is blocked.
	healthyStream := newStream(t, healthyServer.URL+"/video.mkv", int64(contentSize))
	defer healthyStream.Close()

	buffer := make([]byte, 64*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	startTime := time.Now()
	bytesRead, readErr := healthyStream.ReadAt(ctx, buffer, 0)
	elapsed := time.Since(startTime)

	if readErr != nil && readErr != io.EOF {
		t.Fatalf("healthy stream read failed: %v", readErr)
	}
	if bytesRead == 0 {
		t.Fatal("healthy stream read returned 0 bytes")
	}
	if !bytes.Equal(buffer[:bytesRead], content[:bytesRead]) {
		t.Fatal("healthy stream data mismatch")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("healthy stream read took %v — stuck stream is blocking it", elapsed)
	}

	t.Logf("isolation: healthy read completed in %v (%d bytes) while stuck stream was blocked", elapsed, bytesRead)

	// Wait for the stuck goroutine to finish (its context times out at 4s).
	<-stuckDone
}

// ─── Gap 2: WaitForPosition unblocks on HTTP error (EOF marker) ──────────────

// TestWaitForPosition_UnblocksOnHTTPError verifies that when the HTTP server
// returns an error mid-transfer, the transfer's copyData goroutine exits,
// the start() goroutine writes the EOF marker, and WaitForPosition unblocks
// promptly — rather than hanging until context timeout.
func TestWaitForPosition_UnblocksOnHTTPError(t *testing.T) {
	t.Parallel()

	// Server sends exactly 8 KB of data, then closes the connection abruptly.
	// The stream thinks the file is 1 MB, so WaitForPosition for anything past
	// 8 KB must be unblocked by the EOF marker, not by context cancellation.
	bytesBeforeClose := 8 * 1024
	fileSize := int64(1024 * 1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", fileSize-1, fileSize))
		w.WriteHeader(http.StatusPartialContent)

		data := make([]byte, bytesBeforeClose)
		for i := range data {
			data[i] = byte(i % 256)
		}
		w.Write(data)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		// Close without sending the remaining ~1016 KB. The HTTP connection
		// is closed by the handler returning, triggering an unexpected EOF on
		// the client's body.Read.
	}))
	defer server.Close()

	stream := newStream(t, server.URL+"/video.mkv", fileSize)
	defer stream.Close()

	// Read 64 KB starting at offset 0. The server only sends 8 KB and then
	// drops, so the transfer will write EOF marker. WaitForPosition for 64 KB
	// should unblock promptly (within a couple of seconds, not the full 10s
	// timeout).
	buffer := make([]byte, 64*1024)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	startTime := time.Now()
	bytesRead, readErr := stream.ReadAt(ctx, buffer, 0)
	elapsed := time.Since(startTime)

	t.Logf("EOF unblock: bytesRead=%d err=%v elapsed=%v", bytesRead, readErr, elapsed)

	// The critical assertion: the read must NOT have taken the full 10 seconds.
	// If it took close to the context timeout, WaitForPosition was not unblocked
	// by the EOF marker — it was unblocked by context cancellation, which means
	// the EOF marker mechanism is broken.
	if elapsed > 5*time.Second {
		t.Fatalf("WaitForPosition took %v — EOF marker did not unblock it promptly", elapsed)
	}

	// We should get some bytes (at least the 8 KB the server sent) or an error,
	// but never hang.
	if bytesRead > 0 && bytesRead <= bytesBeforeClose {
		// Verify the data we did receive is correct.
		for i := 0; i < bytesRead; i++ {
			if buffer[i] != byte(i%256) {
				t.Fatalf("data mismatch at byte %d: got %d, want %d", i, buffer[i], byte(i%256))
			}
		}
	}
}
