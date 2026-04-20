// Package interfaces defines the core filesystem server abstractions.
package interfaces

// FileSystemServerService is a factory for creating FileSystemServer instances.
type FileSystemServerService interface {
	New(mountpoint string, volumeName string) (FileSystemServer, error)
}

// FileSystemServer represents a mounted FUSE filesystem that can be served
// and gracefully closed. The Serve() method blocks until the filesystem is
// unmounted or an error occurs. Close() triggers graceful shutdown.
type FileSystemServer interface {
	Serve() error
	Close() error
}
