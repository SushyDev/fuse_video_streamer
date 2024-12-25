package main

import (
	"flag"
	"log"
	"os"

	"fuse_video_steamer/fuse"
)

const useVfs = true

func usage() {
	log.Printf("Usage of %s:\n", os.Args[0])
	log.Printf("  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	mountpoint := flag.Arg(0)

	fuseInstance := fuse.New(mountpoint)

	go fuseInstance.Serve()

	done := make(chan bool)
	<-done

	log.Println("Exiting...")
}
