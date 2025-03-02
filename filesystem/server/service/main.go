package service

import "fuse_video_steamer/filesystem/interfaces"

func New(mountpoint string, volumeName string, fileSystemService interfaces.FileSystemServerService) interfaces.FileSystemServer {
	return fileSystemService.New(mountpoint, volumeName)
}
