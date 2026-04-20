// Package interfaces defines the core streaming abstractions used by the FUSE filesystem.
package interfaces

import "context"

// Stream represents an open stream to a remote file. It provides sequential and
// random access to the file contents via HTTP range requests backed by a ring buffer.
// Streams are created by StreamFactory and managed by FUSE file handles.
type Stream interface {
	Identifier() int64
	Size() int64
	URL() string

	ReadAt(ctx context.Context, p []byte, seekPosition int64) (int, error)

	Close() error
	IsClosed() bool
}
