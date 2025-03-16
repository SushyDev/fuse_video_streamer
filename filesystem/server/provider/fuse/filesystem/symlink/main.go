package symlink

import (
	"context"
	"fuse_video_steamer/config"
	"os"
	"path/filepath"

	filesystem_client_interfaces "fuse_video_steamer/filesystem/client/interfaces"

	"github.com/anacrolix/fuse"
)

type Symlink struct {
	client     filesystem_client_interfaces.Client
	identifier uint64
}

func New(client filesystem_client_interfaces.Client, identifier uint64) *Symlink {
	return &Symlink{
		client:     client,
		identifier: identifier,
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
		return "", err
	}

	mountPath := config.GetMountPoint()

	path, err := filepath.Abs(filepath.Join(mountPath, symlink.client.GetName(), linkPath))
	if err != nil {
		return "", err
	}

	return path, nil
}
