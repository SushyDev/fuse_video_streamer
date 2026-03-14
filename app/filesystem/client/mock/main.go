// Package mock provides an in-memory implementation of the filesystem client
// interfaces for use in tests. No gRPC server is required.
package mock

import (
	"fmt"
	"io/fs"
	"sync"
	"sync/atomic"

	interfaces "fuse_video_streamer/filesystem/client/interfaces"
)

// Node is a simple in-memory filesystem node.
type Node struct {
	id         uint64
	name       string
	mode       fs.FileMode
	streamable bool
	size       uint64
}

func NewNode(id uint64, name string, mode fs.FileMode, streamable bool, size uint64) *Node {
	return &Node{
		id:         id,
		name:       name,
		mode:       mode,
		streamable: streamable,
		size:       size,
	}
}

func (n *Node) GetId() uint64        { return n.id }
func (n *Node) GetName() string      { return n.name }
func (n *Node) GetMode() fs.FileMode { return n.mode }
func (n *Node) GetStreamable() bool  { return n.streamable }
func (n *Node) GetSize() uint64      { return n.size }

var _ interfaces.Node = &Node{}

// GetStreamURLFunc is the signature of the injected stream-URL resolver.
type GetStreamURLFunc func(nodeId uint64) (string, error)

// FileSystem is an in-memory implementation of interfaces.FileSystem.
type FileSystem struct {
	mu sync.RWMutex

	root     *Node
	children map[uint64][]*Node

	// getStreamURL controls what GetStreamUrl returns per call.
	// If nil, GetStreamUrl always returns ("", nil).
	getStreamURL GetStreamURLFunc

	// connected mirrors IsConnected behaviour.
	connected atomic.Bool
}

var _ interfaces.FileSystem = &FileSystem{}

// NewFileSystem constructs an in-memory FileSystem.
//
// root is the node returned by Root().
// children maps parent node IDs to their children (used by ReadDirAll).
// getStreamURL is called by GetStreamUrl; pass nil for a no-op.
func NewFileSystem(root *Node, children map[uint64][]*Node, getStreamURL GetStreamURLFunc) *FileSystem {
	fileSystem := &FileSystem{
		root:         root,
		children:     children,
		getStreamURL: getStreamURL,
	}
	fileSystem.connected.Store(true)
	return fileSystem
}

// SetConnected toggles the value returned by the owning Client.IsConnected.
func (fileSystem *FileSystem) SetConnected(v bool) {
	fileSystem.connected.Store(v)
}

func (fileSystem *FileSystem) Root(name string) (interfaces.Node, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()
	if fileSystem.root == nil {
		return nil, fmt.Errorf("root not configured")
	}
	return fileSystem.root, nil
}

func (fileSystem *FileSystem) ReadDirAll(nodeId uint64) ([]interfaces.Node, error) {
	fileSystem.mu.RLock()
	children := fileSystem.children[nodeId]
	fileSystem.mu.RUnlock()

	result := make([]interfaces.Node, len(children))
	for i, child := range children {
		result[i] = child
	}
	return result, nil
}

func (fileSystem *FileSystem) Lookup(parentNodeId uint64, name string) (interfaces.Node, error) {
	fileSystem.mu.RLock()
	children := fileSystem.children[parentNodeId]
	fileSystem.mu.RUnlock()

	for _, child := range children {
		if child.GetName() == name {
			return child, nil
		}
	}
	return nil, fmt.Errorf("node %q not found under %d", name, parentNodeId)
}

func (fileSystem *FileSystem) Remove(parentNodeId uint64, name string) error {
	return nil
}

func (fileSystem *FileSystem) Rename(oldParentNodeId uint64, oldName string, newParentNodeId uint64, newName string) error {
	return nil
}

func (fileSystem *FileSystem) Create(parentNodeId uint64, name string, mode fs.FileMode) error {
	return nil
}

func (fileSystem *FileSystem) MkDir(parentNodeId uint64, name string) (interfaces.Node, error) {
	return nil, nil
}

func (fileSystem *FileSystem) Link(parentNodeId uint64, name string, targetNodeId uint64) error {
	return nil
}

func (fileSystem *FileSystem) ReadFile(nodeId uint64, offset uint64, size uint64) ([]byte, error) {
	return nil, nil
}

func (fileSystem *FileSystem) WriteFile(nodeId uint64, offset uint64, data []byte) (uint64, error) {
	return 0, nil
}

func (fileSystem *FileSystem) GetFileInfo(nodeId uint64) (uint64, fs.FileMode, error) {
	fileSystem.mu.RLock()
	defer fileSystem.mu.RUnlock()

	// Search all children for the node.
	for _, children := range fileSystem.children {
		for _, child := range children {
			if child.GetId() == nodeId {
				return child.GetSize(), child.GetMode(), nil
			}
		}
	}

	// Check root.
	if fileSystem.root != nil && fileSystem.root.GetId() == nodeId {
		return fileSystem.root.GetSize(), fileSystem.root.GetMode(), nil
	}

	return 0, 0, fmt.Errorf("node %d not found", nodeId)
}

func (fileSystem *FileSystem) GetStreamUrl(nodeId uint64) (string, error) {
	fileSystem.mu.RLock()
	fn := fileSystem.getStreamURL
	fileSystem.mu.RUnlock()

	if fn == nil {
		return "", nil
	}
	return fn(nodeId)
}

// Client is an in-memory implementation of interfaces.Client.
type Client struct {
	name       string
	directory  string
	fileSystem *FileSystem
}

var _ interfaces.Client = &Client{}

func NewClient(name string, directory string, fileSystem *FileSystem) *Client {
	return &Client{
		name:       name,
		directory:  directory,
		fileSystem: fileSystem,
	}
}

func (client *Client) GetName() string                      { return client.name }
func (client *Client) GetDirectory() string                 { return client.directory }
func (client *Client) GetFileSystem() interfaces.FileSystem { return client.fileSystem }
func (client *Client) IsConnected() bool                    { return client.fileSystem.connected.Load() }

// ClientRepository is an in-memory implementation of interfaces.ClientRepository.
type ClientRepository struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

var _ interfaces.ClientRepository = &ClientRepository{}

func NewClientRepository(clients ...*Client) *ClientRepository {
	repository := &ClientRepository{
		clients: make(map[string]*Client, len(clients)),
	}
	for _, client := range clients {
		repository.clients[client.GetName()] = client
	}
	return repository
}

func (repository *ClientRepository) GetClientByName(name string) (interfaces.Client, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	client, ok := repository.clients[name]
	if !ok {
		return nil, fmt.Errorf("client %q not found", name)
	}
	return client, nil
}

func (repository *ClientRepository) GetClients() ([]interfaces.Client, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	result := make([]interfaces.Client, 0, len(repository.clients))
	for _, client := range repository.clients {
		result = append(result, client)
	}
	return result, nil
}
