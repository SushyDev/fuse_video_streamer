package interfaces

import "context"

// StreamFactory creates Stream instances for remote files identified by node ID.
// It handles URL resolution, caching, and retry logic for transient failures.
type StreamFactory interface {
	NewStream(ctx context.Context, nodeIdentifier uint64, size uint64) (Stream, error)
	Close() error
}
