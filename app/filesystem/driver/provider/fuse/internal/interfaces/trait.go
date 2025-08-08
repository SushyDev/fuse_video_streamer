package interfaces

import (
	interfaces_filesystem_client "fuse_video_streamer/filesystem/client/interfaces"
)

// Trait

type UseClient interface {
	GetClient() interfaces_filesystem_client.Client
}

type UseIdentifier interface {
	GetIdentifier() uint64
}

type UseRemoteIdentifier interface {
	GetRemoteIdentifier() uint64
}

type UseClosable interface {
	Close() error
	IsClosed() bool
}

type UseSize interface {
	GetSize() uint64
}
