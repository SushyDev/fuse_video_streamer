// Package mock provides an in-memory implementation of interfaces.Stream for
// use in tests. ReadAt behaviour is driven by a queue of canned responses so
// tests are deterministic and do not require time.Sleep.
package mock

import (
	"context"
	"sync"
	"sync/atomic"
)

// ReadAtResponse is one entry in the response queue consumed by ReadAt.
type ReadAtResponse struct {
	// Data is copied into the caller's buffer when non-nil.
	Data []byte

	// Err is returned after (optionally) writing Data.
	Err error

	// Block is closed by the test to unblock a ReadAt that would otherwise hang.
	// If non-nil, ReadAt blocks until Block is closed (or ctx is cancelled)
	// before returning Data/Err.
	Block <-chan struct{}
}

// ReadAtCall records the parameters passed to one ReadAt invocation.
type ReadAtCall struct {
	Offset int64
	Size   int
}

// Stream is an in-memory Stream implementation. Its ReadAt behaviour is
// controlled by enqueueing ReadAtResponse values via EnqueueRead.
type Stream struct {
	identifier int64
	size       int64
	url        string

	mu    sync.Mutex
	queue []ReadAtResponse

	calls []ReadAtCall

	closed atomic.Bool
}

func New(identifier int64, size int64, url string) *Stream {
	return &Stream{
		identifier: identifier,
		size:       size,
		url:        url,
	}
}

// EnqueueRead appends a response to the ReadAt queue. The first call to ReadAt
// dequeues the first response, the second call dequeues the second, and so on.
// If the queue is empty, ReadAt returns (0, nil).
func (stream *Stream) EnqueueRead(response ReadAtResponse) {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	stream.queue = append(stream.queue, response)
}

// Calls returns a copy of all ReadAt invocations recorded so far.
func (stream *Stream) Calls() []ReadAtCall {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	result := make([]ReadAtCall, len(stream.calls))
	copy(result, stream.calls)
	return result
}

func (stream *Stream) Identifier() int64 { return stream.identifier }
func (stream *Stream) Size() int64       { return stream.size }
func (stream *Stream) URL() string       { return stream.url }

func (stream *Stream) ReadAt(ctx context.Context, p []byte, seekPosition int64) (int, error) {
	stream.mu.Lock()

	stream.calls = append(stream.calls, ReadAtCall{Offset: seekPosition, Size: len(p)})

	var response ReadAtResponse
	if len(stream.queue) > 0 {
		response = stream.queue[0]
		stream.queue = stream.queue[1:]
	}

	stream.mu.Unlock()

	// Block if instructed to do so.
	if response.Block != nil {
		select {
		case <-response.Block:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}

	if response.Data != nil {
		n := copy(p, response.Data)
		return n, response.Err
	}

	return 0, response.Err
}

func (stream *Stream) Close() error {
	stream.closed.Store(true)
	return nil
}

func (stream *Stream) IsClosed() bool {
	return stream.closed.Load()
}
