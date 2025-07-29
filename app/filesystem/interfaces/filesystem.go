package interfaces

type FileSystemServerService interface {
	New(mountpoint string, volumeName string) (FileSystemServer, error)
}

type FileSystemServer interface {
	Serve() error
	Close() error
}
