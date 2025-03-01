package filesystem

import "fuse_video_steamer/filesystem/interfaces"

func New(fileSystemService interfaces.FileSystemService, mountpoint string, volumeName string) interfaces.FileSystem {
	return fileSystemService.New(mountpoint, volumeName)
}
