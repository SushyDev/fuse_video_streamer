package streamable

import (
	"context"
	"sync"

	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
	interfaces_logger "fuse_video_streamer/logger/interfaces"

	interfaces_handle "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/handle"
	interfaces_node "fuse_video_streamer/filesystem/driver/provider/fuse/internal/filesystem/node"

	"fuse_video_streamer/filesystem/driver/provider/fuse/internal/pool"
	interfaces_stream "fuse_video_streamer/stream/interfaces"
)

type Service struct {
	node          interfaces_node.StreamableNode
	client        interfaces_filesystem_client.Client
	loggerFactory interfaces_logger.LoggerFactory
	streamFactory interfaces_stream.StreamFactory
	bufferPool    pool.BufferPool

	logger interfaces_logger.Logger

	mu       sync.Mutex
	cond     *sync.Cond
	inflight int
	closed   bool

	ownsStreamFactory bool
}

var _ interfaces_handle.StreamableHandleService = &Service{}

func NewService(
	node interfaces_node.StreamableNode,
	client interfaces_filesystem_client.Client,
	loggerFactory interfaces_logger.LoggerFactory,
	streamFactory interfaces_stream.StreamFactory,
	bufferPool pool.BufferPool,
	logger interfaces_logger.Logger,
	ownsStreamFactory bool,
) *Service {
	s := &Service{
		node:              node,
		client:            client,
		loggerFactory:     loggerFactory,
		streamFactory:     streamFactory,
		bufferPool:        bufferPool,
		logger:            logger,
		ownsStreamFactory: ownsStreamFactory,
	}
	s.cond = sync.NewCond(&s.mu)
	return s
}

func (service *Service) NewHandle(ctx context.Context) (interfaces_handle.StreamableHandle, error) {
	service.mu.Lock()
	if service.closed {
		service.mu.Unlock()
		service.logger.Warn("Attempted to create a new Streamable Handle after service was closed")
		return nil, nil
	}
	service.inflight++
	service.mu.Unlock()

	defer func() {
		service.mu.Lock()
		service.inflight--
		service.cond.Signal()
		service.mu.Unlock()
	}()

	logger, err := service.loggerFactory.NewLogger("File Handle")
	if err != nil {
		service.logger.Error("failed to create logger for Streamable Handle", err)
		return nil, err
	}

	stream, err := service.streamFactory.NewStream(ctx, service.node.GetRemoteIdentifier(), service.node.GetSize())
	if err != nil {
		service.logger.Error("failed to create stream for Streamable Handle", err)
		return nil, err
	}

	return NewHandle(service.node, stream, service.bufferPool, logger), nil
}

func (service *Service) Close() error {
	service.mu.Lock()
	if service.closed {
		service.mu.Unlock()
		return nil
	}
	service.closed = true
	// Wait for all in-flight NewHandle calls to finish before closing the
	// streamFactory, preventing a race between NewStream and Close.
	for service.inflight > 0 {
		service.cond.Wait()
	}
	service.mu.Unlock()

	if service.ownsStreamFactory {
		service.streamFactory.Close()
	}

	return nil
}

func (service *Service) IsClosed() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.closed
}
