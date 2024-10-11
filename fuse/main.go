package fuse

import (
	"flag"
	"log"
    "os"
    filesystem "debrid_drive/filesystem"
)

// usage prints the usage information for the program.
func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

// main is the entry point for the program, handling command-line arguments and mounting the filesystem.
func Start() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 2 {
		usage()
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)

    filesystem.Mount(mountpoint)
}
