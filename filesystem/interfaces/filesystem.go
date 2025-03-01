package interfaces

import (
	"context"
)

type FileSystemService interface {
	New(mountpoint string, volumeName string) FileSystem
}

type FileSystem interface {
	Serve(context.Context)
	Close() error
}
