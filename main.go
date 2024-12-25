package main

import (
	"fuse_video_steamer/config"
	"fuse_video_steamer/fuse"
)

func main() {
	config.Validate()

	mountpoint := config.GetMountPoint()

	fuseInstance := fuse.New(mountpoint)

	go fuseInstance.Serve()

	done := make(chan bool)
	<-done
}
