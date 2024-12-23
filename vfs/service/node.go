package service

import (
	"database/sql"
	"fmt"
	"fuse_video_steamer/vfs/node"
	"sync"
)

type NodeService struct {
	mu sync.RWMutex
}

func NewNodeService() *NodeService {
	return &NodeService{}
}

func GetParentIdentifier(parent *node.Directory) (sql.NullInt64, error) {
	var parentIdentifier sql.NullInt64

	if parent == nil {
		return parentIdentifier, nil
	}

	node := parent.GetNode()

	if node == nil {
		return parentIdentifier, fmt.Errorf("Parent node is nil")
	}

	parentIdentifier.Scan(node.GetIdentifier())

	return parentIdentifier, nil
}

func (service *NodeService) CreateNode(tx *sql.Tx, name string, parent *node.Directory, nodeType node.NodeType) (*uint64, error) {
	existingNodeIdentifier, err := service.FindNode(tx, name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to find node\n%w", err)
	}

	if existingNodeIdentifier != nil {
		return nil, fmt.Errorf("Node already exists")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	query := `
        INSERT INTO nodes (name, parent_id, type)
        VALUES (?, ?, ?)
        RETURNING id
    `

	parentIdentifier, err := GetParentIdentifier(parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to get parent identifier\n%w", err)
	}

	row := tx.QueryRow(query, name, parentIdentifier, nodeType.String())

	var identifier uint64
	err = row.Scan(&identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan node\n%w", err)
	}

	return &identifier, nil
}

func (service *NodeService) UpdateNode(tx *sql.Tx, identifier uint64, name string, parent *node.Directory) error {
	existingNodeIdentifier, err := service.FindNode(tx, name, parent)
	if err != nil {
		return fmt.Errorf("Failed to find node\n%w", err)
	}

	if existingNodeIdentifier != nil && *existingNodeIdentifier != identifier {
		return fmt.Errorf("Node with name %s already exists", name)
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	parentIdentifier, err := GetParentIdentifier(parent)
	if err != nil {
		return fmt.Errorf("Failed to get parent identifier\n%w", err)
	}

	query := `
        UPDATE nodes SET name = ?, parent_id = ?
        WHERE id = ?
    `

	result, err := tx.Exec(query, name, parentIdentifier, identifier)
	if err != nil {
		return fmt.Errorf("Failed to update node\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("No node found with ID: %d", identifier)
	}

	return nil
}

func (service *NodeService) DeleteNode(tx *sql.Tx, identifier uint64) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	query := `
        DELETE FROM nodes
        WHERE id = ?
    `

	result, err := tx.Exec(query, identifier)
	if err != nil {
		return fmt.Errorf("Failed to delete node\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("No node found with ID: %d", identifier)
	}

	return nil
}

func (service *NodeService) FindNode(tx *sql.Tx, name string, parent *node.Directory) (*uint64, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	parentIdentifier, err := GetParentIdentifier(parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to get parent identifier\n%w", err)
	}

	query := `
        SELECT id
        FROM nodes
        WHERE name = ? AND parent_id = ?
    `

	row := tx.QueryRow(query, name, parentIdentifier)

	var identifier uint64
	err = row.Scan(&identifier)
	if err != nil {
		return nil, nil
	}

	return &identifier, nil
}

// --- Helpers

type row interface {
	Scan(dest ...interface{}) error
}
