package service

import (
	"database/sql"
	"fmt"
	vfs_node "fuse_video_steamer/vfs/node"
	"sync"
)

type FileService struct {
	db          *sql.DB
	nodeService *NodeService

	mu sync.RWMutex
}

func NewFileService(db *sql.DB, nodeService *NodeService) *FileService {
	return &FileService{
		db:          db,
		nodeService: nodeService,
	}
}

func (service *FileService) CreateFile(name string, parent *vfs_node.Directory, size uint64, host string) (*uint64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	nodeId, err := service.nodeService.CreateNode(transaction, name, parent, vfs_node.FileNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create node\n%w", err)
	}

	query := `
        INSERT INTO files (node_id, size, host)
        VALUES (?, ?, ?) 
        RETURNING node_id
    `

	row := transaction.QueryRow(query, nodeId, size, host)

	var identifier uint64
	err = row.Scan(&identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to scan file\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("Failed to commit transaction\n%w", err)
	}

	return &identifier, nil
}

func (service *FileService) UpdateFile(nodeId uint64, name string, parent *vfs_node.Directory, size uint64, host string) (*uint64, error) {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction\n%w", err)
	}
	defer transaction.Rollback()

	err = service.nodeService.UpdateNode(transaction, nodeId, name, parent)
	if err != nil {
		return nil, fmt.Errorf("failed to update node\n%w", err)
	}

	query := `
        UPDATE files SET size = ?, host = ?
        WHERE node_id = ? 
        RETURNING node_id
    `

	row := transaction.QueryRow(query, size, host, nodeId)

	var identifier uint64
	err = row.Scan(&identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to scan file\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction\n%w", err)
	}

	return &identifier, nil
}

func (service *FileService) DeleteFile(nodeId uint64) error {
	service.mu.Lock()
	defer service.mu.Unlock()

	transaction, err := service.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction\n%w", err)
	}

	err = service.nodeService.DeleteNode(transaction, nodeId)
	if err != nil {
		return fmt.Errorf("failed to delete node\n%w", err)
	}

	err = transaction.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction\n%w", err)
	}

	return nil
}

func (service *FileService) GetFile(nodeId uint64) (*vfs_node.File, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	query := `
        SELECT n.id, name, parent_id, type, f.size, f.host
        FROM nodes n
        LEFT JOIN files f ON n.id = f.node_id
        WHERE n.id = ? AND type = ?
    `

	row := service.db.QueryRow(query, nodeId, vfs_node.FileNode.String())

	file, err := getFileFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("failed to get file from row\n%w", err)
	}

	return file, nil
}

func (service *FileService) ListFiles() ([]*vfs_node.File, error) {
	query := `
        SELECT id, name, parent_id, type, f.size, f.host
        FROM nodes
        LEFT JOIN files f ON nodes.id = f.node_id
        WHERE type = ?
    `

	rows, err := service.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list files\n%w", err)
	}
	defer rows.Close()

	var files []*vfs_node.File

	for rows.Next() {
		err := rows.Err()
		if err != nil {
			return nil, fmt.Errorf("error occurred during rows iteration\n%w", err)
		}

		file, err := getFileFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to get file from row\n%w", err)
		}

		files = append(files, file)
	}

	return files, nil
}

func (service *FileService) FindFile(name string, parent *vfs_node.Directory) (*vfs_node.File, error) {
	service.mu.RLock()
	defer service.mu.RUnlock()

	var row *sql.Row

	if parent == nil {
		query := `
            SELECT n.id, name, parent_id, type, f.size, f.host
            FROM nodes n
            LEFT JOIN files f ON n.id = f.node_id
            WHERE parent_id IS NULL AND name = ?
        `

		row = service.db.QueryRow(query, name)
	} else {
		query := `
            SELECT n.id, name, parent_id, type, f.size, f.host
            FROM nodes n
            LEFT JOIN files f ON n.id = f.node_id
            WHERE parent_id = ? AND name = ?
        `

		row = service.db.QueryRow(query, parent.GetNode().GetIdentifier(), name)
	}

	file, err := getFileFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("failed to find file by name\n%w", err)
	}

	return file, nil
}

func (service *FileService) GetFiles(parent *vfs_node.Directory) ([]*vfs_node.File, error) {
	var parentIdentifier sql.NullInt64
	if parent != nil {
		parentIdentifier.Scan(parent.GetNode().GetIdentifier())
	}

	query := `
        SELECT n.id, name, parent_id, type, f.size, f.host
        FROM nodes n
        LEFT JOIN files f ON n.id = f.node_id
        WHERE parent_id = ?
    `

	rows, err := service.db.Query(query, parentIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find files by parent\n%w", err)
	}
	defer rows.Close()

	var files []*vfs_node.File

	for rows.Next() {
		err := rows.Err()
		if err != nil {
			return nil, fmt.Errorf("error occurred during rows iteration\n%w", err)
		}

		file, err := getFileFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to get file from row\n%w", err)
		}

		files = append(files, file)
	}

	return files, nil
}

// Defined in directory.go
// type row interface {
// 	Scan(dest ...interface{}) error
// }

func getFileFromRow(row row) (*vfs_node.File, error) {
	var identifier uint64
	var name string
	var parentIdentifier sql.NullInt64
	var nodeTypeStr string
	var size uint64
	var host string

	err := row.Scan(&identifier, &name, &parentIdentifier, &nodeTypeStr, &size, &host)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to scan file\n%w", err)
	}

	var parent_identifier *uint64
	if parentIdentifier.Valid {
		parentIdentifierValue := uint64(parentIdentifier.Int64)
		parent_identifier = &parentIdentifierValue
	}

	nodeType := vfs_node.NodeTypeFromString(nodeTypeStr)

	node := vfs_node.NewNode(identifier, name, parent_identifier, nodeType)

	file := vfs_node.NewFile(node, size, host)

	return file, nil
}
