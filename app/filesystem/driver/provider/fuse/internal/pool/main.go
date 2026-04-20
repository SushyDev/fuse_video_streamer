package pool

import (
	"sync"
)

const (
	SmallVideoBuffer  = int64(64 * 1024 * 1024)   // 64MB for < 1GB files
	MediumVideoBuffer = int64(256 * 1024 * 1024)  // 256MB for 1-10GB files
	LargeVideoBuffer  = int64(512 * 1024 * 1024)  // 512MB for 10GB+ files
	MaxBufferSize     = int64(1024 * 1024 * 1024) // 1GB absolute max
)

type defaultBufferPool struct {
	smallPool  *sync.Pool
	mediumPool *sync.Pool
	largePool  *sync.Pool
	maxPool    *sync.Pool
}

var _ BufferPool = &defaultBufferPool{}

func newDefaultBufferPool() *defaultBufferPool {
	return &defaultBufferPool{
		smallPool: &sync.Pool{
			New: func() any {
				return make([]byte, SmallVideoBuffer)
			},
		},
		mediumPool: &sync.Pool{
			New: func() any {
				return make([]byte, MediumVideoBuffer)
			},
		},
		largePool: &sync.Pool{
			New: func() any {
				return make([]byte, LargeVideoBuffer)
			},
		},
		maxPool: &sync.Pool{
			New: func() any {
				return make([]byte, MaxBufferSize)
			},
		},
	}
}

// NewBufferPool returns a new injectable BufferPool instance.
func NewBufferPool() BufferPool {
	return newDefaultBufferPool()
}

func (bufferPool *defaultBufferPool) Get(size int) []byte {
	bufferSize := calculateBufferSize(int64(size))

	switch bufferSize {
	case SmallVideoBuffer:
		return bufferPool.smallPool.Get().([]byte)
	case MediumVideoBuffer:
		return bufferPool.mediumPool.Get().([]byte)
	case LargeVideoBuffer:
		return bufferPool.largePool.Get().([]byte)
	default:
		return bufferPool.maxPool.Get().([]byte)
	}
}

func (bufferPool *defaultBufferPool) Put(buffer []byte) {
	if buffer == nil {
		return
	}

	bufferSize := int64(len(buffer))

	switch bufferSize {
	case SmallVideoBuffer:
		bufferPool.smallPool.Put(buffer)
	case MediumVideoBuffer:
		bufferPool.mediumPool.Put(buffer)
	case LargeVideoBuffer:
		bufferPool.largePool.Put(buffer)
	case MaxBufferSize:
		bufferPool.maxPool.Put(buffer)
	default:
		// Don't pool buffers of unexpected sizes
		return
	}
}

func calculateBufferSize(fileSize int64) int64 {
	switch {
	case fileSize < 1024*1024*1024: // < 1GB
		return SmallVideoBuffer
	case fileSize < 10*1024*1024*1024: // < 10GB
		return MediumVideoBuffer
	case fileSize < 50*1024*1024*1024: // < 50GB
		return LargeVideoBuffer
	default:
		return MaxBufferSize
	}
}
