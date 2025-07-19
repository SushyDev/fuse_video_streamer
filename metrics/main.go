package metrics

import (
	interfaces_filesystem "fuse_video_streamer/filesystem/interfaces"
)

type Metrics struct {
	fileSystemServer interfaces_filesystem.FileSystemServer
}

func New (
	fileSystemServer interfaces_filesystem.FileSystemServer,
) Metrics {
	return Metrics{
		fileSystemServer: fileSystemServer,
	}
}


func (m Metrics) Serve() {

	m.fileSystemServer.GetMetrics()

}
