package filesystem

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"time"

	interfaces_fuse "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	api "github.com/sushydev/stream_mount_api"
)

type filesystem struct {
	api api.FileSystemServiceClient

	logger interfaces_logger.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces_fuse.FileSystem = &filesystem{}

type node struct {
	id         uint64
	name       string
	mode       io_fs.FileMode
	streamable bool
}

var _ interfaces_fuse.Node = &node{}

func newNode(id uint64, name string, mode io_fs.FileMode, streamable bool) *node {
	return &node{
		id:         id,
		name:       name,
		mode:       mode,
		streamable: streamable,
	}
}

func (n *node) GetId() uint64 {
	return n.id
}

func (n *node) GetName() string {
	return n.name
}

func (n *node) GetMode() io_fs.FileMode {
	return n.mode
}

func (n *node) GetStreamable() bool {
	return n.streamable
}

// convertUnixModeToGoMode converts raw Unix mode bits to Go's io/fs.FileMode
func convertUnixModeToGoMode(unixMode uint32) io_fs.FileMode {
	var mode io_fs.FileMode

	// Extract permission bits (lower 9 bits)
	mode = io_fs.FileMode(unixMode & 0777)

	// Extract and convert file type bits
	switch unixMode & 0170000 { // S_IFMT mask
	case 0040000: // S_IFDIR
		mode |= io_fs.ModeDir
	case 0020000: // S_IFCHR
		mode |= io_fs.ModeDevice | io_fs.ModeCharDevice
	case 0060000: // S_IFBLK
		mode |= io_fs.ModeDevice
	case 0010000: // S_IFIFO
		mode |= io_fs.ModeNamedPipe
	case 0140000: // S_IFSOCK
		mode |= io_fs.ModeSocket
		// Note: Hardlinks are regular files and don't have a special mode bit
		// S_IFLNK (0120000) is for symlinks, which we're not using
	}

	// Extract special bits
	if unixMode&0004000 != 0 { // S_ISUID
		mode |= io_fs.ModeSetuid
	}
	if unixMode&0002000 != 0 { // S_ISGID
		mode |= io_fs.ModeSetgid
	}
	if unixMode&0001000 != 0 { // S_ISVTX
		mode |= io_fs.ModeSticky
	}

	return mode
}

func New(api api.FileSystemServiceClient, logger interfaces_logger.Logger) *filesystem {
	ctx, cancel := context.WithCancel(context.Background())

	return &filesystem{
		api: api,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (fs *filesystem) Root(name string) (interfaces_fuse.Node, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.Root(requestCtx, &api.RootRequest{})
	if err != nil {
		return nil, api.FromResponseError(err)
	}

	root := response.GetRoot()

	return newNode(
		root.GetId(),
		root.GetName(),
		convertUnixModeToGoMode(root.GetMode()),
		root.GetStreamable(),
	), nil
}

func (fs *filesystem) ReadDirAll(nodeId uint64) ([]interfaces_fuse.Node, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.ReadDirAll(requestCtx, &api.ReadDirAllRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return nil, api.FromResponseError(err)
	}

	var nodes []interfaces_fuse.Node
	for _, node := range response.Nodes {
		node := newNode(
			node.GetId(),
			node.GetName(),
			convertUnixModeToGoMode(node.GetMode()),
			node.GetStreamable(),
		)

		nodes = append(nodes, node)
	}

	return nodes, nil

}

func (fs *filesystem) Lookup(parentNodeId uint64, name string) (interfaces_fuse.Node, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.Lookup(requestCtx, &api.LookupRequest{
		NodeId: parentNodeId,
		Name:   name,
	})

	if err != nil {
		return nil, api.FromResponseError(err)
	}

	foundNode := response.GetNode()

	return newNode(
		foundNode.GetId(),
		foundNode.GetName(),
		convertUnixModeToGoMode(foundNode.GetMode()),
		foundNode.GetStreamable(),
	), nil
}

func (fs *filesystem) Remove(parentNodeId uint64, name string) error {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	_, err := fs.api.Remove(requestCtx, &api.RemoveRequest{
		ParentNodeId: parentNodeId,
		Name:         name,
	})

	return api.FromResponseError(err)
}

func (fs *filesystem) Rename(oldParentNodeId uint64, oldName string, newParentNodeId uint64, newName string) error {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	_, err := fs.api.Rename(requestCtx, &api.RenameRequest{
		OldParentNodeId: oldParentNodeId,
		OldName:         oldName,
		NewParentNodeId: newParentNodeId,
		NewName:         newName,
	})

	return api.FromResponseError(err)
}

func (fs *filesystem) Create(parentNodeId uint64, name string, mode io_fs.FileMode) error {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	_, err := fs.api.Create(requestCtx, &api.CreateRequest{
		ParentNodeId: parentNodeId,
		Name:         name,
		Mode:         uint32(mode),
	})

	return api.FromResponseError(err)
}

func (fs *filesystem) MkDir(parentNodeId uint64, name string) (interfaces_fuse.Node, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	fmt.Println("Creating directory:", name, "under parent node ID:", parentNodeId)

	response, err := fs.api.Mkdir(requestCtx, &api.MkdirRequest{
		ParentNodeId: parentNodeId,
		Name:         name,
	})

	if err != nil {
		return nil, api.FromResponseError(err)
	}

	return newNode(
		response.Node.GetId(),
		response.Node.GetName(),
		convertUnixModeToGoMode(response.Node.GetMode()),
		response.Node.GetStreamable(),
	), nil
}

func (fs *filesystem) Link(parentNodeId uint64, name string, targetNodeId uint64) error {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	_, err := fs.api.Link(requestCtx, &api.LinkRequest{
		NodeId:       targetNodeId,
		ParentNodeId: parentNodeId,
		Name:         name,
	})

	return api.FromResponseError(err)
}

func (fs *filesystem) GetFileInfo(nodeId uint64) (uint64, io_fs.FileMode, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.GetFileInfo(requestCtx, &api.GetFileInfoRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return 0, 0, api.FromResponseError(err)
	}

	return response.GetSize(), convertUnixModeToGoMode(response.GetMode()), nil
}

func (fs *filesystem) GetStreamUrl(nodeId uint64) (string, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.GetStreamUrl(requestCtx, &api.GetStreamUrlRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return "", api.FromResponseError(err)
	}

	return response.GetUrl(), nil
}

func (fs *filesystem) ReadFile(nodeId uint64, offset uint64, size uint64) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.ReadFile(requestCtx, &api.ReadFileRequest{
		NodeId: nodeId,
		Offset: offset,
		Size:   size,
	})

	if err != nil {
		return nil, api.FromResponseError(err)
	}

	return response.GetData(), nil
}

func (fs *filesystem) WriteFile(nodeId uint64, offset uint64, data []byte) (uint64, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.WriteFile(requestCtx, &api.WriteFileRequest{
		NodeId: nodeId,
		Offset: offset,
		Data:   data,
	})

	if err != nil {
		return 0, api.FromResponseError(err)
	}

	return response.GetBytesWritten(), nil
}
