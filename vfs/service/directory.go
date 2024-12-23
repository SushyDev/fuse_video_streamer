package service

import (
	"fmt"
	"sync"

	"database/sql"

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
	existingDirectory, err := service.FindDirectory(name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to find directory\n%w", err)
	}

	if existingDirectory != nil {
		return nil, fmt.Errorf("Directory already exists")
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	nodeId, err := service.nodeService.CreateNode(transaction, name, parent, vfs_node.DirectoryNode)
	if err != nil {
		return nil, fmt.Errorf("Failed to create node\n%w", err)
	}

	query := `
        INSERT INTO directories (node_id)
        VALUES (?)
        RETURNING node_id
    `

	row := transaction.QueryRow(query, nodeId)

	var nodeIdentifier uint64
	err = row.Scan(&nodeIdentifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan node\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return &nodeIdentifier, nil
}

func (service *DirectoryService) UpdateDirectory(nodeId uint64, name string, parent *vfs_node.Directory) (*uint64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	err = service.nodeService.UpdateNode(transaction, nodeId, name, parent)
	if err != nil {
		return nil, fmt.Errorf("Failed to update node\n%w", err)
	}

	query := `
        UPDATE directories SET node_id = ?
        WHERE node_id = ?
        RETURNING node_id
    `

	row := transaction.QueryRow(query, nodeId, nodeId)

	var nodeIdentifier uint64
	err = row.Scan(&nodeIdentifier)
	if err != nil {
		return nil, fmt.Errorf("Failed to scan node\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return &nodeIdentifier, err
}

func (service *DirectoryService) DeleteDirectory(nodeId uint64) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return fmt.Errorf("Failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	err = service.nodeService.DeleteNode(transaction, nodeId)
	if err != nil {
		return fmt.Errorf("Failed to delete node\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return nil
}

func (service *DirectoryService) GetDirectory(nodeId uint64) (*vfs_node.Directory, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	query := `
        SELECT node_id, name, parent_id, type
        FROM DIRECTORIES
        LEFT JOIN nodes ON nodes.id = directories.node_id
        WHERE node_id = ? and type = ?
    `

	row := service.db.QueryRow(query, nodeId, vfs_node.DirectoryNode.String())

	directory, err := getDirectoryFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directory from row\n%w", err)
	}

	return directory, nil
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

	directory, err := getDirectoryFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("Failed to get directory from row\n%w", err)
	}

	return directory, nil
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

    node, err := getNodeFromRow(row)
    if err != nil {
        return nil, fmt.Errorf("Failed to get node from row\n%w", err)
    }

    return node, nil
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

	directory := vfs_node.NewDirectory(node)

	return directory, nil
}
