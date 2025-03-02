package interfaces

import (
	"io"
)

type FileSystemServerService interface {
	New(mountpoint string, volumeName string) FileSystemServer
}

type FileSystemServer interface {
	Serve()
	io.Closer
}
