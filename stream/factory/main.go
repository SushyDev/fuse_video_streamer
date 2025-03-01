package factory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fuse_video_steamer/stream"
	"fuse_video_steamer/vfs_api"
)

type Factory struct {
	nodeIdentifier uint64
	size           uint64

	client  vfs_api.FileSystemServiceClient
	streams stream.Map

	mu sync.RWMutex

	context context.Context
	cancel  context.CancelFunc
}

func NewFactory(client vfs_api.FileSystemServiceClient, nodeIdentifier uint64, size uint64) *Factory {
	ctx, cancel := context.WithCancel(context.Background())

	return &Factory{
		nodeIdentifier: nodeIdentifier,
		size:           size,

		client:  client,
		streams: stream.Map{},
		
		context: ctx,
		cancel:  cancel,
	}
}

func (factory *Factory) NewStream(pid uint32) (*stream.Stream, error) {
	factory.mu.Lock()
	defer factory.mu.Unlock()

	if factory.IsClosed() {
		return nil, fmt.Errorf("Factory is closed")
	}

	clientContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := factory.client.GetVideoUrl(clientContext, &vfs_api.GetVideoUrlRequest{
		Identifier: factory.nodeIdentifier,
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to get video url for node with id %d", factory.nodeIdentifier)
	}

	newStream := stream.NewStream(pid, response.Url, int64(factory.size))

	factory.streams.Store(pid, newStream)

	return newStream, nil
}

func (factory *Factory) GetStreamCount() int64 {
	factory.mu.Lock()
	defer factory.mu.Unlock()

	var totalStreams int64

	factory.streams.Range(func(pid uint32, stream *stream.Stream) bool {
		totalStreams++
		return true
	})

	return totalStreams
}

func (factory *Factory) GetStream(pid uint32) (*stream.Stream, bool) {
	factory.mu.Lock()
	defer factory.mu.Unlock()

	existingStream, ok := factory.streams.Load(pid)
	if ok {
		if !existingStream.IsClosed() {
			return existingStream, true
		}

		factory.streams.Delete(pid)
	}

	return nil, false
}

func (factory *Factory) GetOrCreateStream(pid uint32) (*stream.Stream, error) {
	stream, ok := factory.GetStream(pid)
	if ok {
		return stream, nil
	}

	newStream, err := factory.NewStream(pid)
	if err != nil {
		return nil, err
	}

	return newStream, nil
}

func (factory *Factory) DeleteStream(pid uint32) error {
	factory.mu.Lock()
	defer factory.mu.Unlock()

	stream, ok := factory.streams.Load(pid)
	if !ok {
		return nil
	}

	err := stream.Close()
	if err != nil {
		return err
	}

	factory.streams.Delete(pid)

	return nil
}

func (factory *Factory) Close() {
	factory.mu.Lock()

	factory.cancel()

	factory.mu.Unlock()

	var wg sync.WaitGroup

	factory.streams.Range(func(pid uint32, stream *stream.Stream) bool {
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := factory.DeleteStream(pid)
			if err != nil {
				fmt.Println("Failed to close stream:", pid)
			}
		}()

		return true
	})

	wg.Wait()
}

func (factory *Factory) IsClosed() bool {
	select {
	case <-factory.context.Done():
		return true
	default:
		return false
	}
}
