package root

import (
	"context"
	"os"
	"sync"
	"testing"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	mock_client "fuse_video_streamer/filesystem/client/mock"
	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	mock_logger "fuse_video_streamer/logger/mock"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

// stubTree is a minimal Tree for testing the root node.
type stubTree struct {
	mu      sync.Mutex
	counter uint64
}

func (tree *stubTree) GetNextIdentifier() uint64 {
	tree.mu.Lock()
	defer tree.mu.Unlock()
	tree.counter++
	return tree.counter
}

func (tree *stubTree) RegisterNode(_ interfaces_node.AbstractNode) error {
	return nil
}

var _ interfaces_node.Tree = (*stubTree)(nil)

// stubDirectoryNode is a minimal DirectoryNode for testing.
type stubDirectoryNode struct{}

func (n *stubDirectoryNode) GetIdentifier() uint64                          { return 10 }
func (n *stubDirectoryNode) GetRemoteIdentifier() uint64                    { return 10 }
func (n *stubDirectoryNode) GetClient() interfaces_filesystem_client.Client { return nil }
func (n *stubDirectoryNode) GetMode() os.FileMode                           { return os.ModeDir | 0755 }
func (n *stubDirectoryNode) IsClosed() bool                                 { return false }
func (n *stubDirectoryNode) Close() error                                   { return nil }
func (n *stubDirectoryNode) Attr(_ context.Context, _ *fuse.Attr) error     { return nil }
func (n *stubDirectoryNode) Open(_ context.Context, _ *fuse.OpenRequest, _ *fuse.OpenResponse) (fs.Handle, error) {
	return nil, nil
}
func (n *stubDirectoryNode) Lookup(_ context.Context, _ *fuse.LookupRequest, _ *fuse.LookupResponse) (fs.Node, error) {
	return nil, nil
}
func (n *stubDirectoryNode) Remove(_ context.Context, _ *fuse.RemoveRequest) error { return nil }
func (n *stubDirectoryNode) Rename(_ context.Context, _ *fuse.RenameRequest, _ fs.Node) error {
	return nil
}
func (n *stubDirectoryNode) Create(_ context.Context, _ *fuse.CreateRequest, _ *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	return nil, nil, nil
}
func (n *stubDirectoryNode) Mkdir(_ context.Context, _ *fuse.MkdirRequest) (fs.Node, error) {
	return nil, nil
}
func (n *stubDirectoryNode) Link(_ context.Context, _ *fuse.LinkRequest, _ fs.Node) (fs.Node, error) {
	return nil, nil
}

var _ interfaces_node.DirectoryNode = (*stubDirectoryNode)(nil)

// stubDirectoryNodeService is a minimal DirectoryNodeService.
type stubDirectoryNodeService struct {
	closed bool
	mu     sync.Mutex
}

func (service *stubDirectoryNodeService) NewNode(
	_ interfaces_node.DirectoryNode,
	_ uint64,
	_ os.FileMode,
) (interfaces_node.DirectoryNode, error) {
	return &stubDirectoryNode{}, nil
}

func (service *stubDirectoryNodeService) Close() error {
	service.mu.Lock()
	defer service.mu.Unlock()
	service.closed = true
	return nil
}

func (service *stubDirectoryNodeService) IsClosed() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.closed
}

var _ interfaces_node.DirectoryNodeService = (*stubDirectoryNodeService)(nil)

// stubDirectoryNodeServiceFactory creates stubDirectoryNodeService instances.
type stubDirectoryNodeServiceFactory struct{}

func (stubDirectoryNodeServiceFactory) NewService(
	_ interfaces_filesystem_client.Client,
	_ interfaces_node.Tree,
) (interfaces_node.DirectoryNodeService, error) {
	return &stubDirectoryNodeService{}, nil
}

var _ interfaces_node.DirectoryNodeServiceFactory = stubDirectoryNodeServiceFactory{}

// stubDirectoryHandleServiceFactory creates a no-op DirectoryHandleService.
type stubDirectoryHandleService struct{}

func (service *stubDirectoryHandleService) NewHandle(_ interfaces_node.DirectoryNode) (interfaces_handle.DirectoryHandle, error) {
	return nil, nil
}
func (service *stubDirectoryHandleService) Close() error   { return nil }
func (service *stubDirectoryHandleService) IsClosed() bool { return false }

var _ interfaces_handle.DirectoryHandleService = (*stubDirectoryHandleService)(nil)

type stubDirectoryHandleServiceFactory struct{}

func (stubDirectoryHandleServiceFactory) NewService() interfaces_handle.DirectoryHandleService {
	return &stubDirectoryHandleService{}
}

var _ interfaces_handle.DirectoryHandleServiceFactory = stubDirectoryHandleServiceFactory{}

// buildMockClientRepository builds a ClientRepository with n connected clients,
// each advertising a single root node.
func buildMockClientRepository(n int) *mock_client.ClientRepository {
	clients := make([]*mock_client.Client, n)
	for i := 0; i < n; i++ {
		rootNode := mock_client.NewNode(
			uint64(i+1),
			"root",
			os.ModeDir|0755,
			false,
			0,
		)
		fileSystem := mock_client.NewFileSystem(rootNode, nil, nil)
		clients[i] = mock_client.NewClient(
			"client"+string(rune('A'+i)),
			"/tmp",
			fileSystem,
		)
	}
	return mock_client.NewClientRepository(clients...)
}

func newTestRootNode(repo interfaces_filesystem_client.ClientRepository) (*node, error) {
	tree := &stubTree{}
	directoryNodeService := &stubDirectoryNodeService{}

	return NewNode(
		repo,
		stubDirectoryNodeServiceFactory{},
		stubDirectoryHandleServiceFactory{},
		mock_logger.NoopLoggerFactory{},
		directoryNodeService,
		mock_logger.NoopLogger{},
		tree,
	)
}

func lookupRequest(name string) *fuse.LookupRequest {
	return &fuse.LookupRequest{Name: name}
}

// TestLookup_RacesWithClose calls Lookup concurrently with Close.
func TestLookup_RacesWithClose(t *testing.T) {
	t.Parallel()

	repo := buildMockClientRepository(3)
	rootNode, err := newTestRootNode(repo)
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}

	var wg sync.WaitGroup
	const lookups = 50
	wg.Add(lookups + 1)

	for i := 0; i < lookups; i++ {
		go func() {
			defer wg.Done()
			resp := &fuse.LookupResponse{}
			_, _ = rootNode.Lookup(context.Background(), lookupRequest("clientA"), resp)
		}()
	}

	go func() {
		defer wg.Done()
		rootNode.Close()
	}()

	wg.Wait()
}

// TestReadDirAll_RacesWithClose calls ReadDirAll concurrently with Close.
func TestReadDirAll_RacesWithClose(t *testing.T) {
	t.Parallel()

	repo := buildMockClientRepository(3)
	rootNode, err := newTestRootNode(repo)
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}

	var wg sync.WaitGroup
	const readers = 50
	wg.Add(readers + 1)

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			_, _ = rootNode.ReadDirAll(context.Background())
		}()
	}

	go func() {
		defer wg.Done()
		rootNode.Close()
	}()

	wg.Wait()
}

// TestLookup_AfterClose verifies that Lookup after Close returns ENOENT.
func TestLookup_AfterClose(t *testing.T) {
	t.Parallel()

	repo := buildMockClientRepository(1)
	rootNode, err := newTestRootNode(repo)
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}

	rootNode.Close()

	resp := &fuse.LookupResponse{}
	result, err := rootNode.Lookup(context.Background(), lookupRequest("clientA"), resp)

	if err == nil {
		t.Fatal("expected error after Close, got nil")
	}
	if result != nil {
		t.Fatal("expected nil node after Close")
	}
}
