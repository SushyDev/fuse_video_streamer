package node

type NodeType int

const (
	DirectoryNode NodeType = iota
	FileNode
)

func NodeTypeFromString(nodeType string) NodeType {
	switch nodeType {
	case "directory":
		return DirectoryNode
	case "file":
		return FileNode
	default:
		return -1
	}
}

func (n NodeType) String() string {
	return [...]string{
		"directory",
		"file",
	}[n]
}

type Node struct {
	identifier        uint64
	name              string
	parent_identifier *uint64
	node_type         NodeType
}

func NewNode(identifier uint64, name string, parent_identifier *uint64, node_type NodeType) *Node {
	return &Node{
		identifier:        identifier,
		name:              name,
		parent_identifier: parent_identifier,
		node_type:         node_type,
	}
}

func (node *Node) GetIdentifier() uint64 {
	return node.identifier
}

func (node *Node) GetName() string {
	return node.name
}

func (node *Node) GetParentIdentifier() *uint64 {
	return node.parent_identifier
}

func (node *Node) GetType() NodeType {
	return node.node_type
}
