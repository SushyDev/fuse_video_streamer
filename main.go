package main

import (
	"flag"
	"log"
	"os"

	"debrid_drive/api"
	"debrid_drive/fuse"
	"debrid_drive/vfs"
)

const useVfs = true

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT RD_TOKEN\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 2 {
		usage()
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)

	virtualFileSystem := vfs.NewVirtualFileSystem()
	fuseFileSystem := fuse.NewFuseFileSystem(mountpoint, virtualFileSystem)

	go fuseFileSystem.Serve()
	go api.Listen(fuseFileSystem)

	done := make(chan bool)
	<-done
}
