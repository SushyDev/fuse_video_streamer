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

type BufferPool struct {
	smallPool  *sync.Pool
	mediumPool *sync.Pool
	largePool  *sync.Pool
	maxPool    *sync.Pool
}

var globalBufferPool = &BufferPool{
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
func GetBuffer(fileSize int64) []byte {
	bufferSize := calculateBufferSize(fileSize)

	switch bufferSize {
	case SmallVideoBuffer:
		return globalBufferPool.smallPool.Get().([]byte)
	case MediumVideoBuffer:
		return globalBufferPool.mediumPool.Get().([]byte)
	case LargeVideoBuffer:
		return globalBufferPool.largePool.Get().([]byte)
	default:
		return globalBufferPool.maxPool.Get().([]byte)
	}
}

// PutBuffer returns a buffer to the appropriate pool based on its size
func PutBuffer(buffer []byte) {
	if buffer == nil {
		return
	}

	bufferSize := int64(len(buffer))

	switch bufferSize {
	case SmallVideoBuffer:
		globalBufferPool.smallPool.Put(buffer)
	case MediumVideoBuffer:
		globalBufferPool.mediumPool.Put(buffer)
	case LargeVideoBuffer:
		globalBufferPool.largePool.Put(buffer)
	case MaxBufferSize:
		globalBufferPool.maxPool.Put(buffer)
	default:
		// Don't pool buffers of unexpected sizes
		return
	}
}

func calculateBufferSize(fileSize int64) int64 {
	return SmallVideoBuffer

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
