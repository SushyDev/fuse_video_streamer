// Package testutil provides common testing utilities and helpers for the fuse_video_streamer project
package testutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

// TestConfig represents a test configuration for the application
type TestConfig struct {
	MountPoint  string
	VolumeName  string
	FileServers []TestFileServer
}

// TestFileServer represents a test file server configuration
type TestFileServer struct {
	Name   string
	Target string
}

// CreateTempConfig creates a temporary configuration file for testing
func CreateTempConfig(t *testing.T, config TestConfig) string {
	t.Helper()
	
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yml")
	
	configContent := fmt.Sprintf(`mount_point: "%s"
volume_name: "%s"
file_servers:
`, config.MountPoint, config.VolumeName)
	
	for _, server := range config.FileServers {
		configContent += fmt.Sprintf(`  - name: "%s"
    target: "%s"
`, server.Name, server.Target)
	}
	
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	
	return tempDir
}

// WithTempDir executes a test function in a temporary directory
func WithTempDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()
	
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	fn(tempDir)
}

// MockGRPCServer provides a mock gRPC server for testing
type MockGRPCServer struct {
	listener *bufconn.Listener
	server   *grpc.Server
}

// NewMockGRPCServer creates a new mock gRPC server
func NewMockGRPCServer() *MockGRPCServer {
	return &MockGRPCServer{
		listener: bufconn.Listen(1024 * 1024), // 1MB buffer
		server:   grpc.NewServer(),
	}
}

// Start starts the mock gRPC server
func (m *MockGRPCServer) Start() {
	go func() {
		if err := m.server.Serve(m.listener); err != nil {
			// Server stopped, this is expected during testing
		}
	}()
}

// Stop stops the mock gRPC server
func (m *MockGRPCServer) Stop() {
	m.server.Stop()
	m.listener.Close()
}

// Dial creates a client connection to the mock gRPC server
func (m *MockGRPCServer) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return m.listener.Dial()
		}),
		grpc.WithInsecure())
}

// GetServer returns the underlying gRPC server for registering services
func (m *MockGRPCServer) GetServer() *grpc.Server {
	return m.server
}

// TestTimeoutContext creates a context with a reasonable timeout for tests
func TestTimeoutContext(t *testing.T) context.Context {
	t.Helper()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	
	return ctx
}

// ExpectError is a test helper that expects an error and fails if none occurs
func ExpectError(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	
	if err == nil {
		t.Errorf("Expected error with message '%s', but got no error", expectedMessage)
		return
	}
	
	if err.Error() != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, err.Error())
	}
}

// ExpectNoError is a test helper that expects no error and fails if one occurs
func ExpectNoError(t *testing.T, err error) {
	t.Helper()
	
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}
}

// LogCapture captures log output for testing
type LogCapture struct {
	buffer []string
}

// NewLogCapture creates a new log capture instance
func NewLogCapture() *LogCapture {
	return &LogCapture{
		buffer: make([]string, 0),
	}
}

// Write implements io.Writer interface for capturing log output
func (lc *LogCapture) Write(p []byte) (n int, err error) {
	lc.buffer = append(lc.buffer, string(p))
	return len(p), nil
}

// GetLogs returns all captured log messages
func (lc *LogCapture) GetLogs() []string {
	return lc.buffer
}

// Contains checks if any log message contains the specified text
func (lc *LogCapture) Contains(text string) bool {
	for _, log := range lc.buffer {
		if len(log) > 0 && log[:len(log)-1] == text { // Remove trailing newline
			return true
		}
	}
	return false
}

// Clear clears all captured logs
func (lc *LogCapture) Clear() {
	lc.buffer = lc.buffer[:0]
}

// MockReadCloser provides a mock implementation of io.ReadCloser for testing
type MockReadCloser struct {
	content []byte
	pos     int
	closed  bool
}

// NewMockReadCloser creates a new mock ReadCloser with the given content
func NewMockReadCloser(content []byte) *MockReadCloser {
	return &MockReadCloser{
		content: content,
		pos:     0,
		closed:  false,
	}
}

// Read implements io.Reader interface
func (m *MockReadCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.EOF
	}
	
	if m.pos >= len(m.content) {
		return 0, io.EOF
	}
	
	n = copy(p, m.content[m.pos:])
	m.pos += n
	
	return n, nil
}

// Close implements io.Closer interface
func (m *MockReadCloser) Close() error {
	m.closed = true
	return nil
}

// IsClosed returns whether the reader has been closed
func (m *MockReadCloser) IsClosed() bool {
	return m.closed
}

// Reset resets the reader to the beginning
func (m *MockReadCloser) Reset() {
	m.pos = 0
	m.closed = false
}