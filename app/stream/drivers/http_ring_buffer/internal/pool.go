package stream

import (
	"fmt"
	"sync"

	ring_buffer "github.com/sushydev/ring_buffer_go"
)

type BufferPool struct {
	pool *sync.Pool
}

var bufferPoolMap = make(map[int64]*BufferPool)

func NewBufferPool(size int64, startPosition int64) *BufferPool {
	if bufferPool, ok := bufferPoolMap[size]; ok {
		fmt.Println("Buffer pool already exists")
		return bufferPool
	}

	bufferPool := &BufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return ring_buffer.NewLockingRingBuffer(size, startPosition)
			},
		},
	}

	bufferPoolMap[size] = bufferPool

	return bufferPool
}

func (bp *BufferPool) Get() ring_buffer.LockingRingBufferInterface {
	return bp.pool.Get().(ring_buffer.LockingRingBufferInterface)
}

func (bp *BufferPool) Put(buffer ring_buffer.LockingRingBufferInterface) {
	buffer.ResetToPosition(0)
	bp.pool.Put(buffer)
}

func (bp *BufferPool) Close() error {
	bp.pool = nil

	return nil
}
