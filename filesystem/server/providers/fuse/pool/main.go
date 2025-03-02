package pool

import "sync"

type BufferPool struct {
	pool *sync.Pool
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 64*1024*1024)
	},
}

func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

func PutBuffer(buffer []byte) {
	bufferPool.Put(buffer)
}

func CloseBufferPool() {
	bufferPool = sync.Pool{}
}
