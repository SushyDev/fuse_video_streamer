package service

import (
	"database/sql"
	"fmt"
	"sync"

	vfs_node "fuse_video_steamer/vfs/node"
)

type DirectoryService struct {
	db          *sql.DB
	nodeService *NodeService

	mu sync.RWMutex
}

func NewDirectoryService(db *sql.DB, nodeService *NodeService) *DirectoryService {
	return &DirectoryService{
		db:          db,
		nodeService: nodeService,
	}
}

func (service *DirectoryService) CreateDirectory(name string, parent *vfs_node.Directory) (*uint64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	identifier, err := service.nodeService.CreateNode(transaction, name, parent, vfs_node.DirectoryNode)
	if err != nil {
		return nil, fmt.Errorf("Failed to create node\n%w", err)
	}

	query := `
        INSERT INTO directories (node_id)
        VALUES (?)
    `

	result, err := transaction.Exec(query, identifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to insert directory\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected != 1 {
		return nil, fmt.Errorf("Failed to insert directory\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return identifier, nil
}

func (service *DirectoryService) UpdateDirectory(identifier uint64, name string, parent *vfs_node.Directory) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	err = service.nodeService.UpdateNode(transaction, identifier, name, parent)
	if err != nil {
		return fmt.Errorf("Failed to update node\n%w", err)
	}

	query := `
        UPDATE directories SET node_id = ?
        WHERE node_id = ?
    `

	result, err := transaction.Exec(query, identifier, identifier)
	if err != nil {
		return fmt.Errorf("Failed to update directory\n%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Failed to get rows affected\n%w", err)
	}

	if rowsAffected != 1 {
		return fmt.Errorf("Failed to update directory\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return nil
}

func (service *DirectoryService) DeleteDirectory(identifier uint64) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	err = service.nodeService.DeleteNode(transaction, identifier)
	if err != nil {
		return fmt.Errorf("Failed to delete node\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return nil
}

func (service *DirectoryService) GetDirectory(identifier uint64) (*vfs_node.Directory, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	query := `
        SELECT node_id, name, parent_id, type
        FROM DIRECTORIES
        LEFT JOIN nodes ON nodes.id = directories.node_id
        WHERE node_id = ? and type = ?
    `

	row := service.db.QueryRow(query, identifier, vfs_node.DirectoryNode.String())

	return getDirectoryFromRow(row)
}

func (service *DirectoryService) FindDirectory(name string, parent *vfs_node.Directory) (*vfs_node.Directory, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	var row *sql.Row

	if parent == nil {
		query := `
            SELECT nodes.id, name, parent_id, type
            FROM nodes
            LEFT JOIN directories ON directories.node_id = nodes.id
            WHERE name = ? and parent_id IS NULL and type = ?
        `

		row = service.db.QueryRow(query, name, vfs_node.DirectoryNode.String())
	} else {
		query := `
            SELECT nodes.id, name, parent_id, type
            FROM nodes
            LEFT JOIN directories ON directories.node_id = nodes.id
            WHERE name = ? and parent_id = ? and type = ?
        `

		row = service.db.QueryRow(query, name, parent.GetNode().GetIdentifier(), vfs_node.DirectoryNode.String())
	}

	return getDirectoryFromRow(row)
}

func (service *DirectoryService) GetChildNode(name string, parent *vfs_node.Directory) (*vfs_node.Node, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	query := `
        SELECT id, name, parent_id, type
        FROM nodes
        WHERE name = ? AND parent_id = ?
    `

	row := service.db.QueryRow(query, name, parent.GetNode().GetIdentifier())

	return getNodeFromRow(row)
}

func (service *DirectoryService) GetChildNodes(parent *vfs_node.Directory) ([]*vfs_node.Node, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	query := `
        SELECT id, name, parent_id, type
        FROM nodes
        WHERE parent_id = ?
    `

	rows, err := service.db.Query(query, parent.GetNode().GetIdentifier(), vfs_node.DirectoryNode.String())
	if err != nil {
		return nil, fmt.Errorf("Failed to get directories\n%w", err)
	}
	defer rows.Close()

	nodes := make([]*vfs_node.Node, 0)

	for rows.Next() {
		node, err := getNodeFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("Failed to get directory from row\n%w", err)
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// --- Helpers

func getNodeFromRow(row row) (*vfs_node.Node, error) {
	var identifier uint64
	var name string
	var parentIdentifier sql.NullInt64
	var nodeTypeStr string

	err := row.Scan(&identifier, &name, &parentIdentifier, &nodeTypeStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("Failed to scan directory\n%w", err)
	}

	var parentIdentifierPtr *uint64
	if parentIdentifier.Valid {
		parentIdentifierPtr = new(uint64)
		*parentIdentifierPtr = uint64(parentIdentifier.Int64)
	}

	nodeType := vfs_node.NodeTypeFromString(nodeTypeStr)

	node := vfs_node.NewNode(identifier, name, parentIdentifierPtr, nodeType)

	return node, nil
}

func getDirectoryFromRow(row row) (*vfs_node.Directory, error) {
	node, err := getNodeFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to get node from row\n%w", err)
	}

	if node == nil {
		return nil, nil
	}

	directory := vfs_node.NewDirectory(node)

	return directory, nil
}
