package interfaces

import (
	"io/fs"
)

type ClientRepository interface {
	GetClientByName(name string) (Client, error)
	GetClients() ([]Client, error)
}

type Client interface {
	GetName() string
	GetFileSystem() FileSystem
}

type FileSystem interface {
	Root(name string) (Node, error)
	ReadDirAll(nodeId uint64) ([]Node, error)
	Lookup(parentNodeId uint64, name string) (Node, error)
	Remove(parentNodeId uint64, name string) error
	Rename(oldParentNodeId uint64, oldName string, newParentNodeId uint64, newName string) error
	Create(parentNodeId uint64, name string, mode fs.FileMode) error
	MkDir(parentNodeId uint64, name string) (Node, error)
	Link(parentNodeId uint64, name string, targetNodeId uint64) error

	ReadLink(nodeId uint64) (string, error)

	ReadFile(nodeId uint64, offset uint64, size uint64) ([]byte, error)
	WriteFile(nodeId uint64, offset uint64, data []byte) (uint64, error)

	GetFileInfo(nodeId uint64) (size uint64, error error)
	GetStreamUrl(nodeId uint64) (url string, error error)
}

type Node interface {
	GetId() uint64
	GetName() string
	GetMode() fs.FileMode
	GetStreamable() bool
}

