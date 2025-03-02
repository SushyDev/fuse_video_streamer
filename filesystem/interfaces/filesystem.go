package interfaces

import (
	"io"
	"context"
)

type FileSystemServerService interface {
	New(mountpoint string, volumeName string) FileSystemServer
}

type FileSystemServer interface {
	Serve(context.Context)
	io.Closer
}
