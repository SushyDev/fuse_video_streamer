package pool

// BufferPool is the injectable interface for pooled byte-slice allocation.
type BufferPool interface {
	Get(size int) []byte
	Put([]byte)
}
