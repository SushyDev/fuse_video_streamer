package filesystem

import (
	"context"
	"log"
	"os"
	"syscall"
    "fmt"

	"debrid_drive/file"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// FS implements the root filesystem.
type FS struct{}

// Dir represents the root directory.
type Dir struct{}

// Root returns the root directory of the filesystem.
func (FS) Root() (fs.Node, error) {
    fmt.Println("Returning root directory")

	return Dir{}, nil
}

// Attr defines the attributes of the root directory.
func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
    fmt.Println("Setting root directory attributes")

	a.Inode = 1
	a.Mode = os.ModeDir | 0o555 // Read-only directory.

	return nil
}

// Lookup searches for a node in the root directory by name.
func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
    fmt.Println("Looking up file", name)

	if name == "video.mkv" {
		return &file.File{}, nil
	}

	return nil, syscall.Errno(syscall.ENOENT)
}

// ReadDirAll returns all entries in the root directory.
func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
    fmt.Println("Adding video.mkv to file list")

	return []fuse.Dirent{
		{Inode: 2, Name: "video.mkv", Type: fuse.DT_File},
	}, nil
}

func Mount(mountpoint string) {
	// Mount the FUSE filesystem at the specified mount point.
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("videostreamfs"),
		fuse.Subtype("videostreamfs"),
		// fuse.LocalVolume(),
		// fuse.VolumeName("VideoFS"),
	)

	if err != nil {
		log.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer c.Close()

	// Serve the filesystem.
	filesys := FS{}
	err = fs.Serve(c, filesys)
	if err != nil {
		log.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}

	// Check if the mount process has any errors to report.
	// <-c.Ready
	// if err := c.MountError; err != nil {
	// 	log.Fatalf("Mount process encountered an error: %v", err)
	// }
}
