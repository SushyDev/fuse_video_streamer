package fuse

import (
	"debrid_drive/logger"
	"debrid_drive/vfs"
	"fmt"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.FS = &FuseFileSystem{}

type FuseFileSystem struct {
	VFS        *vfs.VirtualFileSystem
    NodeMap   map[uint64]*fs.Node

	connection *fuse.Conn
}

func NewFuseFileSystem(mountpoint string, vfs *vfs.VirtualFileSystem) *FuseFileSystem {
	connection, err := fuse.Mount(
		mountpoint,
		fuse.VolumeName("debrid_drive"),
		fuse.Subtype("debrid_drive"),
		fuse.FSName("debrid_drive"),

		fuse.LocalVolume(),
		fuse.AllowOther(),

		fuse.NoAppleDouble(),
		fuse.NoBrowse(),
	)

	if err != nil {
		logger.Logger.Fatalf("Failed to create FUSE mount: %v", err)
	}

	fuseFileSystem := &FuseFileSystem{
		connection: connection,
		VFS:        vfs,
	}

	return fuseFileSystem
}

func (fileSystem *FuseFileSystem) Root() (fs.Node, error) {
	root := NewDirectoryNode(fileSystem.VFS.Root)

	return root, nil
}

func (fileSystem *FuseFileSystem) Serve() {
	logger.Logger.Info("Serving FUSE filesystem")

	err := fs.Serve(fileSystem.connection, fileSystem)
	if err != nil {
		logger.Logger.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}
}

func (fileSystem *FuseFileSystem) InvalidateEntry(parentID uint64, name string) {
	fileSystem.connection.InvalidateEntry(getNodeID(parentID), name)
}

func (fileSystem *FuseFileSystem) InvalidateNode(ID uint64) {
    fileSystem.connection.InvalidateNode(getNodeID(ID), 0, 0)
}

func (fileSystem *FuseFileSystem) GetNode(ID uint64) (*fs.Node, error) {
    for nodeID, node := range fileSystem.NodeMap {
        if nodeID == ID {
            return node, nil
        }
    }

    return nil, fmt.Errorf("Node with ID %d not found", ID)
}


func getNodeID(ID uint64) fuse.NodeID {
	return fuse.NodeID(ID)
}

