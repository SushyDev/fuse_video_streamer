package factory

import (
	"context"
	"fmt"
	"time"

	"fuse_video_steamer/stream"

	api "github.com/sushydev/stream_mount_api"
)

type Factory struct {
	nodeIdentifier uint64
	size           uint64

	client  api.FileSystemServiceClient

	streams []*stream.Stream

	context context.Context
	cancel  context.CancelFunc
}

func NewFactory(client api.FileSystemServiceClient, nodeIdentifier uint64, size uint64) *Factory {
	ctx, cancel := context.WithCancel(context.Background())

	return &Factory{
		nodeIdentifier: nodeIdentifier,
		size:           size,

		client:  client,
		
		context: ctx,
		cancel:  cancel,
	}
}

func (factory *Factory) NewStream() (*stream.Stream, error) {
	if factory.isClosed() {
		return nil, fmt.Errorf("Factory is closed")
	}

	clientContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := factory.client.GetStreamUrl(clientContext, &api.GetStreamUrlRequest{
		NodeId: factory.nodeIdentifier,
	})

	if err != nil {
		return nil, fmt.Errorf("Failed to get video url for node with id %d", factory.nodeIdentifier)
	}

	newStream := stream.NewStream(response.Url, int64(factory.size))

	factory.streams = append(factory.streams, newStream)

	return newStream, nil
}

func (factory *Factory) Close() {
	factory.cancel()

	for _, stream := range factory.streams {
		stream.Close()
		stream = nil
	}
}

func (factory *Factory) isClosed() bool {
	select {
	case <-factory.context.Done():
		return true
	default:
		return false
	}
}

// TODO - Wont be needed probably
