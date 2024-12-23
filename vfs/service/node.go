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

func (service *NodeService) CreateNode(tx *sql.Tx, name string, parent *node.Directory, nodeType node.NodeType) (*uint64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	query := `
        INSERT INTO nodes (name, parent_id, type)
        VALUES (?, ?, ?)
        RETURNING id
    `

	var parentIdentifier sql.NullInt64
	if parent != nil {
		node := parent.GetNode()

		if node == nil {
			return nil, fmt.Errorf("Parent node is nil")
		}

		parentIdentifier.Scan(node.GetIdentifier())
	}

	row := tx.QueryRow(query, name, parentIdentifier, nodeType.String())

	var identifier uint64
	err := row.Scan(&identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan node\n%w", err)
	}

	return &identifier, nil
}

func (service *NodeService) UpdateNode(tx *sql.Tx, id uint64, name string, parent *node.Directory) error {
	// check if there is already a node with the same name and parent that is not the current node
	existingNodeIdentifier, err := service.FindNode(tx, name, parent)
	if err != nil {
		return fmt.Errorf("Failed to find node\n%w", err)
	}

	if existingNodeIdentifier != nil && *existingNodeIdentifier != id {
		return fmt.Errorf("Node with name %s already exists", name)
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	var parentIdentifier sql.NullInt64
	if parent != nil {
		node := parent.GetNode()

		if node == nil {
			return fmt.Errorf("Parent node is nil")
		}

		parentIdentifier.Scan(node.GetIdentifier())
	}

	query := `
        UPDATE nodes SET name = ?, parent_id = ?
        WHERE id = ?
    `

	result, err := tx.Exec(query, name, parentIdentifier, id)
	if err != nil {
		return fmt.Errorf("Failed to update node\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("No node found with ID: %d", id)
	}

	return nil
}

func (service *NodeService) DeleteNode(tx *sql.Tx, id uint64) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	query := `
        DELETE FROM nodes
        WHERE id = ?
    `

	result, err := tx.Exec(query, id)
	if err != nil {
		return fmt.Errorf("Failed to delete node\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("No node found with ID: %d", id)
	}

	return nil
}

func (service *NodeService) FindNode(tx *sql.Tx, name string, parent *node.Directory) (*uint64, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	var parentIdentifier sql.NullInt64
	if parent != nil {
		node := parent.GetNode()

		if node == nil {
			return nil, fmt.Errorf("Parent node is nil")
		}

		parentIdentifier.Scan(node.GetIdentifier())
	}

	query := `
        SELECT id
        FROM nodes
        WHERE name = ? AND parent_id = ?
    `

	row := tx.QueryRow(query, name, parentIdentifier)

	var identifier uint64
	err := row.Scan(&identifier)
	if err != nil {
		return nil, nil
	}

	return &identifier, nil
}

// func (service *NodeService) GetNodeById(id uint64) (*node.Node, error) {
//     service.mu.RLock()
//     defer service.mu.RUnlock()
//
//     query := `
//         SELECT id, name, parent_id, type
//         FROM nodes
//         WHERE id = ?
//     `
//
//     row := service.db.QueryRow(query, id)
//
//     node, err := scanNode(row)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to scan node\n%w", err)
//     }
//
//     return node, nil
// }

// --- Helpers

type row interface {
	Scan(dest ...interface{}) error
}

// func scanNode(row row) (*node.Node, error) {
//     var identifier uint64
//     var name string
//     var parentIdentifier sql.NullInt64
//     var nodeType string
//
//     err := row.Scan(&identifier, &name, &parentIdentifier, &nodeType)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to scan node\n%w", err)
//     }
//
//     var parentIdentifierPtr *uint64
//     if parentIdentifier.Valid {
//         parentIdentifierPtr = new(uint64)
//         *parentIdentifierPtr = uint64(parentIdentifier.Int64)
//     }
//
//     node := node.NewNode(identifier, name, parentIdentifierPtr, node.NodeTypeFromString(nodeType))
//
//     return node, nil
// }
//
