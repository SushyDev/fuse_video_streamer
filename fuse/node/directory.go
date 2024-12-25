package node

import (
	"context"
	"log"
	"os"
	"sync"
	"syscall"

	"fuse_video_steamer/logger"
	"fuse_video_steamer/vfs_api"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
	"go.uber.org/zap"
)

var _ fs.Handle = &Directory{}

type Directory struct {
	client     vfs_api.FileSystemServiceClient
	identifier uint64

    tempFiles []*TempFile

	logger *zap.SugaredLogger

	mu sync.RWMutex
}

func NewDirectory(client vfs_api.FileSystemServiceClient, identifier uint64) *Directory {
	fuseLogger, _ := logger.GetLogger(logger.FuseLogPath)

	return &Directory{
		client:     client,
		identifier: identifier,
		logger:     fuseLogger,
	}
}

var _ fs.Node = &Directory{}

func (fuseDirectory *Directory) Attr(ctx context.Context, attr *fuse.Attr) error {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	attr.Mode = os.ModeDir | 0o777

	attr.Gid = uint32(os.Getgid())
	attr.Uid = uint32(os.Getuid())

	return nil
}

var _ fs.NodeOpener = &Directory{}

func (fuseDirectory *Directory) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	return fuseDirectory, nil
}

var _ fs.NodeRequestLookuper = &Directory{}

func (fuseDirectory *Directory) Lookup(ctx context.Context, lookupRequest *fuse.LookupRequest, lookupResponse *fuse.LookupResponse) (fs.Node, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	response, err := fuseDirectory.client.Lookup(ctx, &vfs_api.LookupRequest{
		Identifier: fuseDirectory.identifier,
		Name:       lookupRequest.Name,
	})

	if err != nil {
		fuseDirectory.logger.Errorf("Failed to lookup: %v", err)
		return nil, err
	}

    if response.Node == nil {
        return nil, syscall.ENOENT
    }

	switch response.Node.Type {
	case vfs_api.NodeType_FILE:
        sizeResponse, err := fuseDirectory.client.GetVideoSize(ctx, &vfs_api.GetVideoSizeRequest{
            Identifier: response.Node.Identifier,
        })

        if err != nil {
            fuseDirectory.logger.Errorf("Failed to get video size: %v", err)
            return nil, err
        }

        log.Printf("Size: %v\n", sizeResponse.Size)

		return NewFile(fuseDirectory.client, response.Node.Identifier, sizeResponse.Size), nil
	case vfs_api.NodeType_DIRECTORY:
		return NewDirectory(fuseDirectory.client, response.Node.Identifier), nil
	}

    for _, tempFile := range fuseDirectory.tempFiles {
        if tempFile.name == lookupRequest.Name {
            return tempFile, nil
        }
    }

	return nil, syscall.ENOENT
}

var _ fs.HandleReadDirAller = &Directory{}

func (fuseDirectory *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fuseDirectory.mu.RLock()
	defer fuseDirectory.mu.RUnlock()

	response, err := fuseDirectory.client.ReadDirAll(ctx, &vfs_api.ReadDirAllRequest{
		Identifier: fuseDirectory.identifier,
	})

	if err != nil {
		fuseDirectory.logger.Errorf("Failed to read directory: %v", err)
		return nil, err
	}

	var entries []fuse.Dirent

	for _, entry := range response.Nodes {
		switch entry.Type {
		case vfs_api.NodeType_FILE:
			entries = append(entries, fuse.Dirent{
				Name:  entry.Name,
				Type:  fuse.DT_File,
			})
		case vfs_api.NodeType_DIRECTORY:
			entries = append(entries, fuse.Dirent{
				Name:  entry.Name,
				Type:  fuse.DT_Dir,
			})
		}
	}

    for _, tempFile := range fuseDirectory.tempFiles {
        entries = append(entries, fuse.Dirent{
            Name:  tempFile.name,
            Type:  fuse.DT_File,
        })
    }

	return entries, nil
}

var _ fs.NodeRemover = &Directory{}

func (fuseDirectory *Directory) Remove(ctx context.Context, removeRequest *fuse.RemoveRequest) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

    _, err := fuseDirectory.client.Remove(ctx, &vfs_api.RemoveRequest{
        Identifier: fuseDirectory.identifier,
        Name:       removeRequest.Name,
    })

    if err != nil {
        fuseDirectory.logger.Errorf("Failed to remove: %v", err)
        return err
    }

	return nil
}

var _ fs.NodeRenamer = &Directory{}

func (fuseDirectory *Directory) Rename(ctx context.Context, request *fuse.RenameRequest, newDir fs.Node) error {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

    _, err := fuseDirectory.client.Rename(ctx, &vfs_api.RenameRequest{
        ParentIdentifier: fuseDirectory.identifier,
        Name:    request.OldName,
        NewName:    request.NewName,
        NewParentIdentifier:  newDir.(*Directory).identifier,
    })

    if err != nil {
        log.Printf("Failed to rename: %v", err)
        fuseDirectory.logger.Errorf("Failed to rename: %v", err)
        return err
    }

    return nil
}

var _ fs.NodeCreater = &Directory{}

func (fuseDirectory *Directory) Create(ctx context.Context, request *fuse.CreateRequest, response *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

    log.Printf("Create: %s\n", request.Name)

    tempFile := NewTempFile(request.Name, 0)

    fuseDirectory.tempFiles = append(fuseDirectory.tempFiles, tempFile)

    return tempFile, tempFile, nil
}

var _ fs.NodeMkdirer = &Directory{}

func (fuseDirectory *Directory) Mkdir(ctx context.Context, request *fuse.MkdirRequest) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

    response, err := fuseDirectory.client.Mkdir(ctx, &vfs_api.MkdirRequest{
        ParentIdentifier: fuseDirectory.identifier,
        Name:    request.Name,
    })

    if err != nil {
        fuseDirectory.logger.Errorf("Failed to mkdir: %v", err)
        return nil, err
    }

    return NewDirectory(fuseDirectory.client, response.Node.Identifier), nil
}

var _ fs.NodeLinker = &Directory{}

func (fuseDirectory *Directory) Link(ctx context.Context, request *fuse.LinkRequest, oldNode fs.Node) (fs.Node, error) {
	fuseDirectory.mu.Lock()
	defer fuseDirectory.mu.Unlock()

    oldFile := oldNode.(*File)

    _, err := fuseDirectory.client.Link(ctx, &vfs_api.LinkRequest{
        Identifier: oldFile.identifier,
        ParentIdentifier: fuseDirectory.identifier,
        Name:    request.NewName,
    })

    if err != nil {
        fuseDirectory.logger.Errorf("Failed to link: %v", err)
        return nil, err
    }

    return oldFile, nil
}
