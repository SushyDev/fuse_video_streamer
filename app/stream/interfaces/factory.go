package interfaces

import "context"

type StreamFactory interface {
	NewStream(ctx context.Context, nodeIdentifier uint64, size uint64) (Stream, error)
	Close() error
}
