package interfaces

type FileSystemServerService interface {
	New(mountpoint string, volumeName string) FileSystemServer
}

type FileSystemServer interface {
	Serve()
	Close() error
}
