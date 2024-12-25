package node

import (
	"context"
	"os"
	"sync"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

var _ fs.Handle = &File{}
var _ fs.HandleReleaser = &File{}

type TempFile struct {
	name string
	size uint64
	data []byte

	mu sync.RWMutex
}

var _ fs.Node = &TempFile{}

func (tempFile *TempFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	tempFile.mu.RLock()
	defer tempFile.mu.RUnlock()

	attr.Mode = 0o777
	attr.Size = tempFile.size

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

var _ fs.NodeOpener = &File{}

func (tempFile *TempFile) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	openResponse.Flags |= fuse.OpenKeepCache

	return tempFile, nil
}

func NewTempFile(name string, size uint64) *TempFile {
	return &TempFile{
		name: name,
		size: size,
	}
}

var _ fs.HandleReader = &TempFile{}

func (tempFile *TempFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	tempFile.mu.RLock()
	defer tempFile.mu.RUnlock()

	resp.Data = tempFile.data

	return nil
}

var _ fs.HandleWriter = &TempFile{}

func (tempFile *TempFile) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	tempFile.mu.Lock()
	defer tempFile.mu.Unlock()

	tempFile.data = req.Data
	tempFile.size = uint64(len(req.Data))

	resp.Size = len(req.Data)

	return nil
}
