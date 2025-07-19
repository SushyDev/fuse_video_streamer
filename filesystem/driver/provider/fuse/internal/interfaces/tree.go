package interfaces

type Tree interface {
	GetNextIdentifier() uint64
	RegisterNodeOnIdentifier(identifier uint64, node Node) error
}
