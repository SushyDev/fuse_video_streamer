package symlink

import (
	"context"
	"fmt"
	"fuse_video_streamer/config"
	"os"
	"path/filepath"
	"syscall"

	intefaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"github.com/anacrolix/fuse"
)

type Symlink struct {
	client     intefaces_filesystem_client.Client
	identifier uint64

	logger interfaces_logger.Logger
}

func New(client intefaces_filesystem_client.Client, logger interfaces_logger.Logger, identifier uint64) *Symlink {
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
		message := fmt.Sprintf("Failed to read symlink with identifier %d and path %s", symlink.identifier, linkPath)
		symlink.logger.Error(message, err)

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
