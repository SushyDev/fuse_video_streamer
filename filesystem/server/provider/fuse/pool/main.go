package pool

import (
	"sync"
)

const (
	SmallReadBuffer   = int64(4 * 1024)           // 4KB for small reads
	SmallVideoBuffer  = int64(64 * 1024 * 1024)   // 64MB for < 1GB files
	MediumVideoBuffer = int64(256 * 1024 * 1024)  // 256MB for 1-10GB files
	LargeVideoBuffer  = int64(512 * 1024 * 1024)  // 512MB for 10GB+ files
	MaxBufferSize     = int64(1024 * 1024 * 1024) // 1GB absolute max
)

// GetOptimalBufferSize returns an optimal buffer size for file operations
func GetOptimalBufferSize(fileSize int64) int64 {
	return calculateBufferSize(fileSize)
}

type BufferPool struct {
	smallReadPool *sync.Pool
	smallPool     *sync.Pool
	mediumPool    *sync.Pool
	largePool     *sync.Pool
	maxPool       *sync.Pool
}

var globalBufferPool = &BufferPool{
	smallReadPool: &sync.Pool{
		New: func() any {
			return make([]byte, SmallReadBuffer)
		},
	},
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

// GetBuffer returns a buffer appropriate for the given file size
func GetBuffer(bufferSize int64) []byte {
	switch {
	case bufferSize <= SmallReadBuffer:
		return globalBufferPool.smallReadPool.Get().([]byte)[:bufferSize]
	case bufferSize <= SmallVideoBuffer:
		return globalBufferPool.smallPool.Get().([]byte)[:bufferSize]
	case bufferSize <= MediumVideoBuffer:
		return globalBufferPool.mediumPool.Get().([]byte)[:bufferSize]
	case bufferSize <= LargeVideoBuffer:
		return globalBufferPool.largePool.Get().([]byte)[:bufferSize]
	default:
		return globalBufferPool.maxPool.Get().([]byte)[:bufferSize]
	}
}

// PutBuffer returns a buffer to the appropriate pool based on its capacity
func PutBuffer(buffer []byte) {
	if buffer == nil {
		return
	}
	
	bufferCap := int64(cap(buffer))
	
	switch bufferCap {
	case SmallReadBuffer:
		// Reset the slice to full capacity before pooling
		globalBufferPool.smallReadPool.Put(buffer[:cap(buffer)])
	case SmallVideoBuffer:
		globalBufferPool.smallPool.Put(buffer[:cap(buffer)])
	case MediumVideoBuffer:
		globalBufferPool.mediumPool.Put(buffer[:cap(buffer)])
	case LargeVideoBuffer:
		globalBufferPool.largePool.Put(buffer[:cap(buffer)])
	case MaxBufferSize:
		globalBufferPool.maxPool.Put(buffer[:cap(buffer)])
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
