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
	url          string
	seekPosition int64
	context      context.Context
	cancel       context.CancelFunc

	body io.ReadCloser

	mu sync.RWMutex
}

func NewConnection(url string, seekPosition int64) (*Connection, error) {
	if seekPosition < 0 {
		return nil, fmt.Errorf("Invalid seek position: %d", seekPosition)
	}

	connectionContext, connectionCancel := context.WithCancel(context.Background())

	connection := &Connection{
		url:          url,
		seekPosition: seekPosition,
		context:      connectionContext,
		cancel:       connectionCancel,
	}

	return connection, nil
}

func (connection *Connection) Read(buf []byte) (int, error) {
	connection.mu.RLock()
	defer connection.mu.RUnlock()

	if connection.IsClosed() {
		return 0, context.Canceled
	}

	if connection.body != nil {
		return connection.body.Read(buf)
	}

	request, err := http.NewRequestWithContext(connection.context, "GET", connection.url, nil)
	if err != nil {
		return 0, fmt.Errorf("Failed to create request")
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", connection.seekPosition)
	request.Header.Set("Range", rangeHeader)

	fmt.Println("Requesting", rangeHeader)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1,
			MaxConnsPerHost:     1,
			MaxIdleConnsPerHost: 1,
			Proxy:               http.ProxyFromEnvironment,
		},
		Timeout: 6 * time.Hour,
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("Failed to do request: %v", err)
	}

	if response.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("Failed to get partial content: %d", response.StatusCode)
	}

	connection.body = response.Body

	return response.Body.Read(buf)
}

func (connection *Connection) GetSeekPosition() int64 {
	return connection.seekPosition
}

func (connection *Connection) IsClosed() bool {
	select {
	case <-connection.context.Done():
		return true
	default:
	}

	return false
}

func (connection *Connection) Close() error {
	if connection.body != nil {
		connection.body.Close()
	}

	connection.cancel()

	connection.body = nil

	return nil
}
