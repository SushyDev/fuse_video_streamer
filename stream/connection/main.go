package connection

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var _ io.ReadCloser = &Connection{}

// Shared HTTP client for connection reuse
var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(100),
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxConnsPerHost:       20,  // Increased for better parallelism
		MaxIdleConnsPerHost:   10,  // Increased to maintain more connections
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    true,
		Proxy:                 http.ProxyFromEnvironment,
		WriteBufferSize:       256 * 1024, // 256KB write buffer
		ReadBufferSize:        256 * 1024, // 256KB read buffer
	},
	Timeout: 4 * time.Hour,
}

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
	// Check closed state once at the beginning
	if connection.closed.Load() {
		return 0, nil
	}

	connection.mu.RLock()
	body := connection.body
	connection.mu.RUnlock()

	if body != nil {
		return body.Read(buf)
	}

	connection.mu.Lock()
	defer connection.mu.Unlock()

	// Double-check pattern - check again under write lock
	if connection.closed.Load() {
		return 0, nil
	}

	// Check if another goroutine already created the body
	if connection.body != nil {
		return connection.body.Read(buf)
	}

	request, err := http.NewRequestWithContext(connection.context, "GET", connection.url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request")
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", connection.startPosition)
	request.Header.Set("Range", rangeHeader)

	response, err := sharedHTTPClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %v", err)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %v", err)
	}

	// Some systems like zurg use 200 status code for partial content
	if response.StatusCode != http.StatusPartialContent && response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get partial content: %d", response.StatusCode)
	}

	connection.body = response.Body

	return response.Body.Read(buf)
}

func (connection *Connection) Close() error {
	if !connection.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	connection.cancel()

	if connection.body != nil {
		err := connection.body.Close()
		if err != nil {
			return fmt.Errorf("error closing body: %v", err)
		}

		connection.body = nil
	}

	return nil
}

func (connection *Connection) isClosed() bool {
	return connection.closed.Load()
}
