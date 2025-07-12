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

	if connection.closed.Load() {
		return 0, nil
	}

	request, err := http.NewRequestWithContext(connection.context, "GET", connection.url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request")
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", connection.startPosition)
	request.Header.Set("Range", rangeHeader)

	client := &http.Client{
		Transport: &http.Transport{
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
		},
		Timeout: 4 * time.Hour,
	}

	response, err := client.Do(request)
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
