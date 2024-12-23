package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func New() (*sql.DB, error) {
	db, err := initializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize database: %v", err)
	}

	return db, nil
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
