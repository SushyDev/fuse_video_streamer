package connection

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
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
}

func NewConnection(url string, startPosition int64) (*Connection, error) {
	if startPosition < 0 {
		return nil, fmt.Errorf("Invalid seek position: %d", startPosition)
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
	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.isClosed() {
		return 0, nil
	}

	if connection.body != nil {
		return connection.body.Read(buf)
	}

	request, err := http.NewRequestWithContext(connection.context, "GET", connection.url, nil)
	if err != nil {
		return 0, fmt.Errorf("Failed to create request")
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", connection.startPosition)
	request.Header.Set("Range", rangeHeader)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			MaxIdleConnsPerHost: 1,
			Proxy:               http.ProxyFromEnvironment,
		},
		Timeout: 4 * time.Hour,
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("Failed to do request: %v", err)
	}

	// Some systems like zurg use 200 status code for partial content
	if response.StatusCode != http.StatusPartialContent && response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Failed to get partial content: %d", response.StatusCode)
	}

	connection.body = response.Body

	return response.Body.Read(buf)
}

func (connection *Connection) Close() error {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.isClosed() {
		return nil
	}

	connection.cancel()

	if connection.body != nil {
		err := connection.body.Close()
		if err != nil {
			return fmt.Errorf("Error closing body: %v", err)
		}

		connection.body = nil
	}

	return nil
}

func (connection *Connection) isClosed() bool {
	select {
	case <-connection.context.Done():
		return true
	default:
		return false
	}
}
