// Package mock provides a trivial BufferPool implementation for tests.
// It uses make([]byte, size) directly — no pooling — and tracks live allocations
// with an atomic counter so tests can assert there are no leaks after a handle closes.
package mock

import "sync/atomic"

// BufferPool is an injectable, allocation-tracking BufferPool.
type BufferPool struct {
	// live is incremented on Get and decremented on Put.
	live atomic.Int64
}

// Get allocates a fresh byte slice of exactly size bytes and increments the
// live-allocation counter.
func (pool *BufferPool) Get(size int) []byte {
	pool.live.Add(1)
	return make([]byte, size)
}

// Put decrements the live-allocation counter. The buffer itself is discarded.
func (pool *BufferPool) Put(_ []byte) {
	pool.live.Add(-1)
}

// LiveAllocations returns the number of buffers that have been obtained via Get
// but not yet returned via Put.
func (pool *BufferPool) LiveAllocations() int64 {
	return pool.live.Load()
}
