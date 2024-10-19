package vfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"

	"debrid_drive/logger"
	"debrid_drive/real_debrid"
	"debrid_drive/vlc"
)

type File struct {
	name         string
	iNode        uint64
	videoUrl     string
	videoStreams sync.Map // map[uint64]*vlc.Stream // map of video streams per PID
	mu           sync.RWMutex
	chunks       int64
	size         int64
}

func (file *File) Attr(ctx context.Context, a *fuse.Attr) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	a.Size = uint64(file.size)
	a.Inode = file.iNode
	a.Mode = os.ModePerm

	a.Atime = time.Unix(0, 0)
	a.Mtime = time.Unix(0, 0)
	a.Ctime = time.Unix(0, 0)
	// a.Valid = 1

	return nil
}

func (file *File) Remove(ctx context.Context) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	logger.Logger.Infof("Removing file %s", file.name)

	return nil
}

func (file *File) Open(ctx context.Context, openRequest *fuse.OpenRequest, openResponse *fuse.OpenResponse) (fs.Handle, error) {
	logger.Logger.Infof("Opening file %s - %d", file.name, file.size)

	// openResponse.Flags |= fuse.OpenDirectIO

	return file, nil
}

func (file *File) Release(ctx context.Context, releaseRequest *fuse.ReleaseRequest) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	logger.Logger.Infof("Releasing file %s", file.name)

	file.videoStreams.Range(func(key, value interface{}) bool {
		stream := value.(*vlc.Stream)
		stream.Close()

		return true
	})

	file.videoStreams.Clear()

	return nil
}

func (file *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readResponse *fuse.ReadResponse) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	// fmt.Printf("Reading %d bytes at offset %d\n", readRequest.Size, readRequest.Offset)

	if readRequest.Dir {
		return fmt.Errorf("read request is for a directory")
	}

	stream, err := file.getVideoStream(readRequest.Pid)
	if err != nil {
		return fmt.Errorf("failed to get video stream: %w", err)
	}

	// start, bufferSize, err := file.calculateReadBoundaries(readRequest.Offset, int64(readRequest.Size))
	// if err != nil {
	// 	return fmt.Errorf("failed to calculate read boundaries: %w", err)
	// }

	_, err = stream.Seek(readRequest.Offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek in video stream: %w", err)
	}

	buffer := make([]byte, readRequest.Size)
	bytesRead, err := stream.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read from video stream: %w", err)
	}

	readResponse.Data = buffer[:bytesRead]

	return nil
}

func (file *File) Flush(ctx context.Context, flushRequest *fuse.FlushRequest) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	logger.Logger.Infof("Flushing file %s", file.name)

	// stream, err := file.getVideoStream(flushRequest.Pid)
	// if err != nil {
	//     return fmt.Errorf("failed to get video stream: %w", err)
	// }
	//
	// if stream != nil {
	//     err := stream.Close()
	//     if err != nil {
	//         return fmt.Errorf("failed to close video stream: %w", err)
	//     }
	// }

	return nil
}

// --- Helpers ---

func (file *File) getVideoStream(pid uint32) (*vlc.Stream, error) {
	stream, ok := file.videoStreams.Load(pid)
	if ok {
		return stream.(*vlc.Stream), nil
	}

	unrestrictedFile, err := real_debrid.UnrestrictLink(file.videoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to unrestrict link: %w", err)
	}

	videoStream := vlc.NewStream(unrestrictedFile.Download, file.size)

	file.videoStreams.Store(pid, videoStream)

	return videoStream, nil
}

func (file *File) calculateReadBoundaries(start int64, requestedSize int64) (int64, int64, error) {
	if start >= file.size {
		return 0, 0, io.EOF
	}

	if start+requestedSize > file.size {
		requestedSize = file.size - start
	}

	return start, requestedSize, nil
}
