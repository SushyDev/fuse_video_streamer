package app

import (
	"flag"
	"fmt"
	"log"
	"os"

	"debrid_drive/database"
	"debrid_drive/logger"
	"debrid_drive/real_debrid"
	"debrid_drive/vfs"
)

const useVfs = true

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT RD_TOKEN\n", os.Args[0])
	flag.PrintDefaults()
}

func Start() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 2 {
		usage()
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)
	addFileRequest := make(chan vfs.AddFileRequest)

	if useVfs {
		go vfs.Mount(mountpoint, addFileRequest)
	}

	InitDatabase()
	logger.Logger.Info("Database initialized")

	IndexDebrid()
	logger.Logger.Info("Debrid indexed")

	files, err := database.GetAllDebridFiles()
	if err != nil {
		logger.Logger.Errorf("Error getting all debrid files", err)
	}

	for _, file := range files {
		path := file.TorrentId + file.Path

		addFileRequest <- vfs.AddFileRequest{
			Path:     path,
			VideoUrl: file.Link,
			Size:     file.Bytes,
		}
	}

	logger.Logger.Info("Files added to VFS")

	done := make(chan bool)
	<-done
}

func InitDatabase() {
	database.Start()
}

func IndexDebrid() {
	torrentResponse, err := real_debrid.GetTorrents()
	if err != nil {
		fmt.Println("Error getting torrents:", err)
		return
	}

	fmt.Println("Torrents:", len(*torrentResponse))

	for _, torrent := range *torrentResponse {
		torrentInDatabase, _ := database.GetTorrentByTorrentId(torrent.ID)

		if torrentInDatabase != nil {
			fmt.Println("Torrent in database:", torrent.ID)
			continue // Torrent is already in the database
		}

		fmt.Println("Torrent not in database:", torrent.ID)

		indexTorrent(torrent)
	}
}

var playableExtensions = []string{".mkv", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm"}

func hasPlayableFile(files []real_debrid.File) bool {
	for _, file := range files {
		for _, ext := range playableExtensions {
			if ext == file.Path[len(file.Path)-len(ext):] {
				return true
			}
		}
	}

	return false
}

func indexTorrent(torrent real_debrid.Torrent) {
	torrentInfo, err := real_debrid.GetTorrentInfo(torrent.ID)
	if err != nil {
		fmt.Println("Error getting torrent info:", err)
		return
	}

	var playable int
	if yes := hasPlayableFile(torrentInfo.Files); yes {
		playable = 1
	} else {
		playable = 0
	}

	insert := database.RealDebridTorrent{
		TorrentId: torrent.ID,
		Filename:  torrent.Filename,
		Bytes:     torrent.Bytes,
		Host:      torrent.Host,
		Split:     torrent.Split,
		Added:     torrent.Added,
		Ended:     torrent.Ended,
		Playable:  playable,
	}

	err = database.InsertRealDebridTorrent(insert)
	if err != nil {
		fmt.Println("Error inserting torrent into database:", torrent.ID, err)
		return
	}

	if insert.Playable == 0 {
		fmt.Println("Torrent not playable:", torrent.ID)

		return
	}

	filesSkipped := 0

	for index, file := range (*torrentInfo).Files {
		if file.Selected != 1 {
			filesSkipped++
			continue
		}

		file := database.RealDebridFile{
			TorrentId: torrent.ID,
			FileId:    file.ID,
			Path:      file.Path,
			Bytes:     file.Bytes,
			Link:      torrent.Links[index-filesSkipped],
		}

		err = database.InsertRealDebridFile(file)
		if err != nil {
			logger.Logger.Errorf("Error inserting file into database", err)
			continue
		}

		fmt.Println("File:", file)
	}
}
