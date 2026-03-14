// Package mock provides shared test configuration helpers.
package mock

import (
	"fuse_video_streamer/config"
)

// MinimalConfig returns a Config with disk cache disabled, suitable for tests
// that do not require a real mount point.
func MinimalConfig() *config.Config {
	return &config.Config{
		MountPoint:      "/mnt/test",
		VolumeName:      "test",
		EnableDiskCache: false,
	}
}
