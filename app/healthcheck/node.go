package healthcheck

import (
	"context"
	"sync/atomic"
	"syscall"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

const (
	HealthCheckFileName = "healthcheck.txt"
	HealthCheckContent  = "Ok"
)

// HealthCheckFileNode is a virtual in-memory file that responds to health checks
type HealthCheckFileNode struct {
	content string
	size    uint64
	closed  atomic.Bool
}

// NewHealthCheckFileNode creates a new virtual health check file node
func NewHealthCheckFileNode() *HealthCheckFileNode {
	return &HealthCheckFileNode{
		content: HealthCheckContent,
		size:    uint64(len(HealthCheckContent)),
	}
}

// Attr returns the file attributes for the health check file
func (node *HealthCheckFileNode) Attr(ctx context.Context, attr *fuse.Attr) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	attr.Mode = 0o444 // read-only
	attr.Size = node.size

	return nil
}

// Open opens the health check file for reading
func (node *HealthCheckFileNode) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	if node.IsClosed() {
		return nil, syscall.ENOENT
	}

	// Only allow reading
	if openRequest.Flags&fuse.OpenWriteOnly != 0 || openRequest.Flags&fuse.OpenReadWrite != 0 {
		return nil, syscall.EACCES
	}

	return node, nil
}

// Read reads from the health check file
func (node *HealthCheckFileNode) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	if node.IsClosed() {
		return syscall.ENOENT
	}

	// Return the content starting from the requested offset
	data := []byte(node.content)
	if readRequest.Offset >= int64(len(data)) {
		readResponse.Data = []byte{}
		return nil
	}

	end := readRequest.Offset + int64(readRequest.Size)
	if end > int64(len(data)) {
		end = int64(len(data))
	}

	readResponse.Data = data[readRequest.Offset:end]
	return nil
}

// Release is called when the file handle is closed
func (node *HealthCheckFileNode) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	return nil
}

// Close marks the node as closed
func (node *HealthCheckFileNode) Close() error {
	node.closed.Store(true)
	return nil
}

// IsClosed returns whether the node has been closed
func (node *HealthCheckFileNode) IsClosed() bool {
	return node.closed.Load()
}
