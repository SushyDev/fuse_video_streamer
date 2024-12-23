package node

type Directory struct {
	node *Node
}

func NewDirectory(node *Node) *Directory {
	return &Directory{
		node: node,
	}
}

func (directory *Directory) GetNode() *Node {
	return directory.node
}
