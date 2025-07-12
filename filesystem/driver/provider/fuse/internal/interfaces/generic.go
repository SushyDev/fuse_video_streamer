package interfaces

type useClosable interface {
	Close() error
	IsClosed() bool
}
