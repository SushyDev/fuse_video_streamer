package vfs

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT VIDEO_URL\n", os.Args[0])
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

	Mount(mountpoint)
}

func Mount(mountpoint string) {
	fileSystem := &FileSystem{
		files: make(map[string]*File),
	}

	videoUrl := flag.Arg(1)

	fmt.Println("Video URL:", videoUrl)

	// Add the initial file to the filesystem
	fileSystem.files["video.mkv"] = &File{
		VideoUrl: videoUrl,
	}

	channel, err := fuse.Mount(
		mountpoint,
		fuse.FSName("videostreamfs"),
		fuse.Subtype("videostreamfs"),
		fuse.AllowOther(),
	)
	if err != nil {
		log.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer channel.Close()

	err = fs.Serve(channel, fileSystem)
	if err != nil {
		log.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}

	// Check if the mount process has any errors to report.
	// <-c.Ready
	// if err := c.MountError; err != nil {
	// 	log.Fatalf("Mount process encountered an error: %v", err)
	// }
}
