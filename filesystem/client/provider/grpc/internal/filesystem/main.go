package filesystem

import (
	"fmt"
	"context"
	io_fs "io/fs"
	"time"

	"fuse_video_streamer/filesystem/client/interfaces"
	"fuse_video_streamer/logger"

	api "github.com/sushydev/stream_mount_api"
)

type filesystem struct {
	api api.FileSystemServiceClient

	logger *logger.Logger

	ctx    context.Context
	cancel context.CancelFunc
}

var _ interfaces.FileSystem = &filesystem{}

type node struct {
	id         uint64
	name       string
	mode       io_fs.FileMode
	streamable bool
}

var _ interfaces.Node = &node{}

func newNode(id uint64, name string, mode io_fs.FileMode, streamable bool) *node {
	return &node{
		id:   id,
		name: name,
		mode: mode,
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

func New(api api.FileSystemServiceClient, logger *logger.Logger) *filesystem {
	ctx, cancel := context.WithCancel(context.Background())

	return &filesystem{
		api: api,

		logger: logger,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (fs *filesystem) Root(name string) (interfaces.Node, error) {
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
		io_fs.FileMode(root.GetMode()),
		root.GetStreamable(),
	), nil
}

func (fs *filesystem) ReadDirAll(nodeId uint64) ([]interfaces.Node, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.ReadDirAll(requestCtx, &api.ReadDirAllRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return nil, api.FromResponseError(err)
	}

	var nodes []interfaces.Node
	for _, node := range response.Nodes {
		node := newNode(
			node.GetId(),
			node.GetName(),
			io_fs.FileMode(node.GetMode()),
			node.GetStreamable(),
		)

		nodes = append(nodes, node)
	}

	return nodes, nil

}

func (fs *filesystem) Lookup(parentNodeId uint64, name string) (interfaces.Node, error) {
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
		io_fs.FileMode(foundNode.GetMode()),
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

func (fs *filesystem) MkDir(parentNodeId uint64, name string) (interfaces.Node, error) {
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
		io_fs.FileMode(response.Node.GetMode()),
		response.Node.GetStreamable(),
	), nil
}


func (fs *filesystem) Link(parentNodeId uint64, name string, targetNodeId uint64) error {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	_, err := fs.api.Link(requestCtx, &api.LinkRequest{
		NodeId: targetNodeId,
		ParentNodeId: parentNodeId,
		Name:         name,
	})

	return api.FromResponseError(err)
}

func (fs *filesystem) ReadLink(nodeId uint64) (string, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.ReadLink(requestCtx, &api.ReadLinkRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return "", api.FromResponseError(err)
	}

	return response.GetPath(), nil
}

func (fs *filesystem) GetFileInfo(nodeId uint64) (uint64, error) {
	requestCtx, cancel := context.WithTimeout(fs.ctx, 10*time.Second)
	defer cancel()

	response, err := fs.api.GetFileInfo(requestCtx, &api.GetFileInfoRequest{
		NodeId: nodeId,
	})

	if err != nil {
		return 0, api.FromResponseError(err)
	}

	return response.GetSize(), nil
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
		Size: size,
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
		Data: data,
	})

	if err != nil {
		return 0, api.FromResponseError(err)
	}

	return response.GetBytesWritten(), nil
}
