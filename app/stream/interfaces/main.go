package interfaces

import "context"

type Stream interface {
	Identifier() int64
	Size() int64
	URL() string

	ReadAt(ctx context.Context, p []byte, seekPosition int64) (int, error)

	Close() error
	IsClosed() bool
}
