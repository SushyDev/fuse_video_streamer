package interfaces

type useIdentifier interface {
	GetIdentifier() uint64
}

type useRemoteIdentifier interface {
	GetRemoteIdentifier() uint64
}

type useClosable interface {
	Close() error
	IsClosed() bool
}
