package symlink

import (
	"context"
	"fuse_video_streamer/config"
	"fuse_video_streamer/logger"
	"os"
	"path/filepath"
	"syscall"

	filesystem_client_interfaces "fuse_video_streamer/filesystem/client/interfaces"

	"github.com/anacrolix/fuse"
)

type Symlink struct {
	client     filesystem_client_interfaces.Client
	identifier uint64

	logger *logger.Logger
}

func New(client filesystem_client_interfaces.Client, identifier uint64) *Symlink {
	logger, err := logger.NewLogger("Symlink Node")
	if err != nil {
		panic(err)
	}

	return &Symlink{
		client:     client,
		identifier: identifier,
		
		logger: logger,
	}
}

func (symlink *Symlink) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = os.ModeSymlink

	return nil
}

func (symlink *Symlink) Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error) {
	fileSystem := symlink.client.GetFileSystem()

	linkPath, err := fileSystem.ReadLink(symlink.identifier)
	if err != nil {
		symlink.logger.Error("Failed to read symlink", err)

		return "", syscall.ENOENT
	}

	mountPath := config.GetMountPoint()

	path, err := filepath.Abs(filepath.Join(mountPath, symlink.client.GetName(), linkPath))
	if err != nil {
		symlink.logger.Error("Failed to get absolute path for symlink", err)

		return "", syscall.ENOENT
	}

	return path, nil
}
