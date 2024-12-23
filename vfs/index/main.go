package index

import (
	"database/sql"
	"fmt"

	// vfs_node "fuse_video_steamer/vfs/node"

	_ "modernc.org/sqlite"
)

type Index struct {
	db *sql.DB
}

func New() (*Index, error) {
	db, err := initializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize database: %v", err)
	}

	index := &Index{
		db: db,
	}

	return index, nil
}

func initializeDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite", "./example.db")
	if err != nil {
		return nil, fmt.Errorf("Failed to open database: %v", err)
	}

	_, err = db.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return nil, fmt.Errorf("Failed to enable foreign keys: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS nodes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            parent_id INTEGER,
            type TEXT NOT NULL CHECK(type IN ('directory', 'file')),

            FOREIGN KEY(parent_id) REFERENCES nodes(id) ON DELETE CASCADE
        );
    `)
	if err != nil {
		return nil, fmt.Errorf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS directories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            node_id INTEGER NOT NULL,

            FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
        );
    `)
	if err != nil {
		return nil, fmt.Errorf("Failed to create table: %v", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS files (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            node_id INTEGER NOT NULL,
            size INTEGER NOT NULL,
            host TEXT NOT NULL,

            FOREIGN KEY(node_id) REFERENCES nodes(id) ON DELETE CASCADE
        );
    `)
	if err != nil {
		return nil, fmt.Errorf("Failed to create table: %v", err)
	}

	return db, nil
}

func (index *Index) GetDB() *sql.DB {
	return index.db
}

// --- Directory

// func (index *Index) RegisterDirectory(parentIdentifier sql.NullInt64, name string) (uint64, error) {
// 	result, err := index.db.Exec(`INSERT INTO directories (name, parent_id) VALUES (?, ?, ?);`, name, parentIdentifier)
// 	if err != nil {
// 		return 0, fmt.Errorf("Failed to register directory: %v", err)
// 	}
//
//     identifier, err := result.LastInsertId()
//     if err != nil {
//         return 0, fmt.Errorf("Failed to get identifier: %v", err)
//     }
//
// 	return uint64(identifier), nil
// }
//
// func (index *Index) DeregisterDirectory(directory *vfs_node.Directory) error {
// 	_, err := index.db.Exec(`DELETE FROM directories WHERE id = ?;`, directory.GetIdentifier())
// 	if err != nil {
// 		return fmt.Errorf("Failed to deregister directory: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) UpdateDirectory(directory *vfs_node.Directory) error {
// 	_, err := index.db.Exec(`UPDATE directories SET name = ?, parent_id = ? WHERE id = ?;`,
// 		directory.GetName(),
// 		*directory.GetParentIdentifier(),
// 		directory.GetIdentifier(),
// 	)
// 	if err != nil {
// 		return fmt.Errorf("Failed to update directory: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) GetDirectory(identifier uint64) (*vfs_node.Directory, error) {
// 	row := index.db.QueryRow(`SELECT id, name, parent_id FROM directories WHERE id = ?;`, identifier)
//
// 	var directoryIdentifier uint64
//     var directoryName string
//     var directoryParentIdentifier sql.NullInt64
//
// 	err := row.Scan(&directoryIdentifier, &directoryName, &directoryParentIdentifier)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to get directory: %v", err)
// 	}
//
//     var parent *vfs_node.Directory
//
//     if directoryParentIdentifier.Valid {
//         parent, err = index.GetDirectory(uint64(directoryParentIdentifier.Int64))
//
//         if err != nil {
//             return nil, fmt.Errorf("Failed to get parent directory: %v", err)
//         }
//     }
//
//     directory := vfs_node.NewDirectory(directoryIdentifier, directoryName, parent)
//
// 	return directory, nil
// }
//
// func (index *Index) FindDirectory(name string) (*vfs_node.Directory, error) {
// 	row := index.db.QueryRow(`SELECT id, name, parent_id FROM directories WHERE name = ?;`, name)
//
// 	var directory *vfs_node.Directory
//
// 	err := row.Scan(directory)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to find directory: %v", err)
// 	}
//
// 	return directory, nil
// }
//
// // --- File
//
// func (index *Index) RegisterFile(file *vfs_node.File) error {
// 	_, err := index.db.Exec(`INSERT INTO files (name, parent_id size, host, video_url) VALUES (?, ?, ?, ?, ?, ?);`,
// 		file.GetIdentifier(),
// 		file.GetName(),
// 		*file.GetParentIdentifier(),
// 		file.GetSize(),
// 		file.GetHost(),
// 		file.GetVideoURL(),
// 	)
//
// 	if err != nil {
// 		return fmt.Errorf("Failed to register file: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) DeregisterFile(file *vfs_node.File) error {
// 	_, err := index.db.Exec(`DELETE FROM files WHERE id = ?;`, file.GetIdentifier())
// 	if err != nil {
// 		return fmt.Errorf("Failed to deregister file: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) RenameFile(file *vfs_node.File, name string) error {
// 	_, err := index.db.Exec(`UPDATE files SET name = ? WHERE id = ?;`, name, file.GetIdentifier())
// 	if err != nil {
// 		return fmt.Errorf("Failed to rename file: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) UpdateFile(file *vfs_node.File) error {
// 	_, err := index.db.Exec(`UPDATE files SET name = ?, parent_id = ?, size = ?, host = ?, video_url = ? WHERE id = ?;`,
// 		file.GetName(),
// 		*file.GetParentIdentifier(),
// 		file.GetSize(),
// 		file.GetHost(),
// 		file.GetVideoURL(),
// 		file.GetIdentifier(),
// 	)
//
// 	if err != nil {
// 		return fmt.Errorf("Failed to update file: %v", err)
// 	}
//
// 	return nil
// }
//
// func (index *Index) GetFile(identifier uint64) (*vfs_node.File, error) {
//     row := index.db.QueryRow(`SELECT id, name, parent_id size, host, video_url FROM files WHERE id = ?;`, identifier)
//
//     var file *vfs_node.File
//
//     err := row.Scan(file)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to get file: %v", err)
//     }
//
//     return file, nil
// }
//
// func (index *Index) FindFile(name string) (*vfs_node.File, error) {
//     row := index.db.QueryRow(`SELECT id, name, parent_id size, host, video_url FROM files WHERE name = ?;`, name)
//
//     var file *vfs_node.File
//
//     err := row.Scan(file)
//     if err != nil {
//         return nil, fmt.Errorf("Failed to find file: %v", err)
//     }
//
//     return file, nil
// }
