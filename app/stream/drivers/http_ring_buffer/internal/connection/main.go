package connection

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var _ io.ReadCloser = &Connection{}

// sharedTransport is a package-level HTTP transport shared across all connections
// to enable TCP connection pooling and avoid leaking idle connections.
var sharedTransport = &http.Transport{
	TLSClientConfig: &tls.Config{
		ClientSessionCache: tls.NewLRUClientSessionCache(100),
	},
	ForceAttemptHTTP2:   true,
	MaxIdleConns:        100,
	MaxConnsPerHost:     10,
	MaxIdleConnsPerHost: 3,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  true,
	Proxy:               http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout:  60 * time.Second,
}

// readTimeout is the maximum time to wait for a single Read() call on the
// HTTP response body before considering the connection stalled.
const readTimeout = 5 * time.Minute

type Connection struct {
	url           string
	startPosition int64

	context context.Context
	cancel  context.CancelFunc

	body io.ReadCloser

	mu sync.RWMutex

	closed atomic.Bool
}

func NewConnection(url string, startPosition int64) (*Connection, error) {
	if startPosition < 0 {
		return nil, fmt.Errorf("invalid seek position: %d", startPosition)
	}

	connectionContext, connectionCancel := context.WithCancel(context.Background())

	connection := &Connection{
		url:           url,
		startPosition: startPosition,
		context:       connectionContext,
		cancel:        connectionCancel,
	}

	return connection, nil
}

func (connection *Connection) Read(buf []byte) (int, error) {
	if connection.isClosed() {
		return 0, io.EOF
	}

	connection.mu.RLock()
	body := connection.body
	connection.mu.RUnlock()

	if body != nil {
		return connection.readWithTimeout(body, buf)
	}

	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.isClosed() {
		return 0, io.EOF
	}

	// Double-check after acquiring write lock.
	if connection.body != nil {
		return connection.readWithTimeout(connection.body, buf)
	}

	request, err := http.NewRequestWithContext(connection.context, "GET", connection.url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", connection.startPosition)
	request.Header.Set("Range", rangeHeader)

	client := &http.Client{
		Transport: sharedTransport,
		// Remove the 4-hour blanket timeout. We use per-read timeouts
		// and context cancellation instead for finer control.
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %v", err)
	}

	// Some systems like zurg use 200 status code for partial content
	if response.StatusCode != http.StatusPartialContent && response.StatusCode != http.StatusOK {
		response.Body.Close()
		return 0, fmt.Errorf("failed to get partial content: %d", response.StatusCode)
	}

	connection.body = response.Body

	return connection.readWithTimeout(response.Body, buf)
}

// readWithTimeout wraps a body.Read with a per-read deadline so that stalled
// TCP connections (zero bytes/sec, no FIN/RST) are detected within readTimeout
// rather than hanging for hours.
func (connection *Connection) readWithTimeout(body io.ReadCloser, buf []byte) (int, error) {
	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1)
	go func() {
		n, err := body.Read(buf)
		ch <- result{n, err}
	}()

	timer := time.NewTimer(readTimeout)
	defer timer.Stop()

	select {
	case res := <-ch:
		return res.n, res.err
	case <-timer.C:
		// Read timed out - close connection to unblock the body.Read goroutine.
		connection.cancel()
		if body != nil {
			body.Close()
		}
		return 0, fmt.Errorf("read timed out after %v", readTimeout)
	case <-connection.context.Done():
		return 0, connection.context.Err()
	}
}

func (connection *Connection) Close() error {
	if !connection.closed.CompareAndSwap(false, true) {
		return nil
	}

	connection.cancel()

	connection.mu.Lock()
	body := connection.body
	connection.body = nil
	connection.mu.Unlock()

	if body != nil {
		if err := body.Close(); err != nil {
			return fmt.Errorf("error closing body: %v", err)
		}
	}

	return nil
}

func (connection *Connection) isClosed() bool {
	return connection.closed.Load()
}
