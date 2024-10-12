package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var Database *sql.DB

type RealDebridTorrent struct {
	ID        string
	TorrentId string
	Filename  string
	Bytes     int64
	Host      string
	Split     int
	Added     string
	Ended     string
	Playable  int
}

type RealDebridFile struct {
	ID        int
	TorrentId string
	FileId    int
	Path      string
	Bytes     int64
	Link      string
}

func Start() {
	var err error
	Database, err = sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	// defer Database.Close()

	fmt.Println("Database started")

	_, err = Database.Exec(`
                CREATE TABLE IF NOT EXISTS real_debrid_torrents (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        torrent_id TEXT UNIQUE,
                        filename TEXT,
                        bytes INTEGER,
                        host TEXT,
                        split INTEGER,
                        added TEXT,
                        ended TEXT,
                        playable INTEGER
                )
        `)
	if err != nil {
		log.Fatal(err)
	}

	_, err = Database.Exec(`
                CREATE TABLE IF NOT EXISTS real_debrid_files (
                        id INTEGER PRIMARY KEY AUTOINCREMENT,
                        torrent_id TEXT,
                        file_id INTEGER,
                        path TEXT,
                        bytes INTEGER,
                        link TEXT,

                        FOREIGN KEY(torrent_id) REFERENCES real_debrid_torrents(torrent_id)
                )
        `)
	if err != nil {
		log.Fatal(err)
	}
}

func GetTorrentByTorrentId(id string) (*RealDebridTorrent, error) {
	torrent := &RealDebridTorrent{}

	err := Database.QueryRow("SELECT torrent_id, filename, bytes, host, split, added, ended, playable FROM real_debrid_torrents WHERE torrent_id = ?", id).Scan(&torrent.TorrentId, &torrent.Filename, &torrent.Bytes, &torrent.Host, &torrent.Split, &torrent.Added, &torrent.Ended, &torrent.Playable)
	if err != nil {
		return nil, err
	}

	return torrent, nil
}

func InsertRealDebridTorrent(torrent RealDebridTorrent) error {
	_, err := Database.Exec("INSERT INTO real_debrid_torrents (torrent_id, filename, bytes, host, split, added, ended, playable) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", torrent.TorrentId, torrent.Filename, torrent.Bytes, torrent.Host, torrent.Split, torrent.Added, torrent.Ended, torrent.Playable)
	if err != nil {
		return err
	}

	return nil
}

func GetRealDebridFile(torrentId string, fileId int) (*RealDebridFile, error) {
	file := &RealDebridFile{}

	err := Database.QueryRow("SELECT id, torrent_id, file_id, path, bytes, link FROM real_debrid_files WHERE torrent_id = ? AND file_id = ?", torrentId, fileId).Scan(&file.ID, &file.TorrentId, &file.FileId, &file.Path, &file.Bytes, &file.Link)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func InsertRealDebridFile(file RealDebridFile) error {
	_, err := Database.Exec("INSERT INTO real_debrid_files (torrent_id, file_id, path, bytes, link) VALUES (?, ?, ?, ?, ?)", file.TorrentId, file.FileId, file.Path, file.Bytes, file.Link)
	if err != nil {
		return err
	}

	return nil
}

func GetAllDebridFiles() ([]RealDebridFile, error) {
	rows, err := Database.Query("SELECT id, torrent_id, file_id, path, bytes, link FROM real_debrid_files")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []RealDebridFile

	for rows.Next() {
		var file RealDebridFile

		err := rows.Scan(&file.ID, &file.TorrentId, &file.FileId, &file.Path, &file.Bytes, &file.Link)
		if err != nil {
			return nil, err
		}

		files = append(files, file)
	}

	return files, nil
}
