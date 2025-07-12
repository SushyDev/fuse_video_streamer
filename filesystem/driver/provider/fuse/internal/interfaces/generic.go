package interfaces

type useIntentifier interface {
	GetIdentifier() uint64
}

type useClosable interface {
	Close() error
	IsClosed() bool
}
