// Package fuse contains integration tests for the FUSE node/handle layer.
// No real FUSE filesystem is mounted; the test exercises Open, Read, and
// Release directly on the node structs.
package fuse_test

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	mock_client "fuse_video_streamer/filesystem/client/mock"
	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"
	mock_pool "fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool/mock"
	mock_logger "fuse_video_streamer/logger/mock"
	interfaces_stream "fuse_video_streamer/stream/interfaces"
	mock_stream "fuse_video_streamer/stream/mock"

	handle_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle/streamable"
	node_streamable "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node/streamable"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	"github.com/anacrolix/fuse"
	"go.uber.org/goleak"
)

// integrationStreamFactory is a StreamFactory that hands out mock streams and
// supports injecting failures.
type integrationStreamFactory struct {
	mu       sync.Mutex
	failNext bool
}

func (factory *integrationStreamFactory) NewStream(_ context.Context, nodeIdentifier uint64, size uint64) (interfaces_stream.Stream, error) {
	factory.mu.Lock()
	fail := factory.failNext
	factory.failNext = false
	factory.mu.Unlock()

	if fail {
		return nil, context.DeadlineExceeded
	}

	stream := mock_stream.New(int64(nodeIdentifier), int64(size), "http://example.com")
	// Pre-populate responses so reads return data without blocking.
	for i := 0; i < 50; i++ {
		stream.EnqueueRead(mock_stream.ReadAtResponse{Data: []byte("integration_data")})
	}
	return stream, nil
}

func (factory *integrationStreamFactory) Close() error { return nil }

func (factory *integrationStreamFactory) InjectFailure() {
	factory.mu.Lock()
	factory.failNext = true
	factory.mu.Unlock()
}

var _ interfaces_stream.StreamFactory = (*integrationStreamFactory)(nil)

// integrationHandleServiceFactory wires the integration stream factory into the
// streamable handle service.
type integrationHandleServiceFactory struct {
	streamFactory interfaces_stream.StreamFactory
	bufferPool    *mock_pool.BufferPool
	loggerFactory interfaces_logger.LoggerFactory
}

func (factory *integrationHandleServiceFactory) NewService(
	node interfaces_node.StreamableNode,
	client interfaces_filesystem_client.Client,
) (interfaces_handle.StreamableHandleService, error) {
	logger, err := factory.loggerFactory.NewLogger("Streamable Service")
	if err != nil {
		return nil, err
	}
	return handle_streamable.NewService(
		node,
		client,
		factory.loggerFactory,
		factory.streamFactory,
		factory.bufferPool,
		logger,
	), nil
}

var _ interfaces_handle.StreamableHandleServiceFactory = (*integrationHandleServiceFactory)(nil)

// buildStreamableNode creates a StreamableNode backed by a mock client.
func buildStreamableNode(
	nodeID uint64,
	fileSize uint64,
	handleFactory interfaces_handle.StreamableHandleServiceFactory,
) (*node_streamable.Node, error) {
	child := mock_client.NewNode(nodeID, "video.mkv", os.FileMode(0644), true, fileSize)
	children := map[uint64][]*mock_client.Node{
		1: {child},
	}
	fileSystem := mock_client.NewFileSystem(
		mock_client.NewNode(1, "root", os.ModeDir|0755, false, 0),
		children,
		nil,
	)
	client := mock_client.NewClient("test", "/tmp", fileSystem)

	logger, _ := mock_logger.NoopLoggerFactory{}.NewLogger("Streamable Node")

	return node_streamable.NewNode(
		client,
		handleFactory,
		logger,
		nodeID,
		nodeID,
		fileSize,
		os.FileMode(0644),
	)
}

// TestFuseIntegration_ConcurrentOpenReadRelease constructs 20 streamable nodes
// and hammers them with 50 concurrent goroutines for 3 seconds, mixing reads,
// closes, and injected stream errors. The race detector and goleak are both
// active.
func TestFuseIntegration_ConcurrentOpenReadRelease(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		nodeCount  = 20
		goroutines = 50
		duration   = 3 * time.Second
		fileSize   = uint64(10 * 1024)
	)

	streamFactory := &integrationStreamFactory{}
	bufferPool := &mock_pool.BufferPool{}

	handleFactory := &integrationHandleServiceFactory{
		streamFactory: streamFactory,
		bufferPool:    bufferPool,
		loggerFactory: mock_logger.NoopLoggerFactory{},
	}

	nodes := make([]*node_streamable.Node, nodeCount)
	for i := range nodes {
		node, err := buildStreamableNode(uint64(i+1), fileSize, handleFactory)
		if err != nil {
			t.Fatalf("buildStreamableNode %d: %v", i, err)
		}
		nodes[i] = node
	}
	defer func() {
		for _, node := range nodes {
			node.Close()
		}
	}()

	deadline := time.Now().Add(duration)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()

			rng := rand.New(rand.NewSource(time.Now().UnixNano()))

			for time.Now().Before(deadline) {
				nodeIndex := rng.Intn(nodeCount)
				node := nodes[nodeIndex]

				if node.IsClosed() {
					continue
				}

				// Occasionally inject a stream error.
				if rng.Intn(10) == 0 {
					streamFactory.InjectFailure()
				}

				openReq := &fuse.OpenRequest{}
				openResp := &fuse.OpenResponse{}

				handle, err := node.Open(context.Background(), openReq, openResp)
				if err != nil || handle == nil {
					continue
				}

				streamableHandle, ok := handle.(interfaces_handle.StreamableHandle)
				if !ok {
					continue
				}

				// Read at random offsets 1-10 times.
				reads := 1 + rng.Intn(10)
				for r := 0; r < reads; r++ {
					offset := int64(rng.Intn(int(fileSize)))
					readReq := &fuse.ReadRequest{
						Offset: offset,
						Size:   64,
					}
					readResp := &fuse.ReadResponse{}
					_ = streamableHandle.Read(context.Background(), readReq, readResp)
				}

				releaseReq := &fuse.ReleaseRequest{}
				_ = streamableHandle.Release(context.Background(), releaseReq)
			}
		}()
	}

	wg.Wait()

	// Give any in-flight goroutines time to drain before goleak checks.
	time.Sleep(100 * time.Millisecond)
}
