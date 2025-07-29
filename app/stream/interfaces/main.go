package interfaces

type Stream interface {
	Identifier() int64
	Size() int64
	Url() string

	ReadAt(p []byte, seekPosition int64) (int, error)

	Close() error
	IsClosed() bool
}
