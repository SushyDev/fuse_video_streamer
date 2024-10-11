package vfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"bazil.org/fuse"

	"debrid_drive/stream"
)

type File struct {
	VideoUrl    string                // URL of the video file
	VideoStream *stream.PartialReader // Holds the open connection stream
	mu          sync.Mutex            // Mutex for thread safety
}

// Attr is called to retrieve the attributes (metadata) of the file.
func (file *File) Attr(ctx context.Context, a *fuse.Attr) error {
	if file.VideoStream == nil {
		var err error

		file.VideoStream, err = stream.NewPartialReader(file.VideoUrl) // Initialize the video stream
		if err != nil {
			return fmt.Errorf("failed to initialize video stream: %w", err)
		}
	}

	a.Mode = os.ModePerm
	a.Size = uint64(file.VideoStream.Size) // Set the size of the file
	// a.Blocks = (uint64(f.VideoStream.Size) + 511) / 512 // File system blocks
	a.Atime = time.Now()
	a.Mtime = time.Now()
	a.Ctime = time.Now()

	return nil
}

func (file *File) Remove(ctx context.Context) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	fmt.Println("Removing file")

	file.VideoStream.Close()

	return nil
}

func (file *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readReponse *fuse.ReadResponse) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	start, bufferSize, err := file.calculateReadBoundaries(readRequest.Offset, int64(readRequest.Size))
	if err != nil {
		return err
	}

	if err := file.seekVideoStream(start); err != nil {
		return err
	}

	buffer, totalBytesRead, err := file.readFromVideoStream(bufferSize)
	if err != nil {
		return err
	}

	file.setResponseData(readReponse, buffer, totalBytesRead)

	return nil
}

func (file *File) calculateReadBoundaries(start, requestedSize int64) (int64, int64, error) {
	fileSize := file.VideoStream.Size

	if start >= fileSize {
		return 0, 0, fmt.Errorf("read position is beyond the end of the file")
	}

	if start+requestedSize > fileSize {
		remainingSize := fileSize - start

		requestedSize = remainingSize
	}

	return start, requestedSize, nil
}

func (file *File) seekVideoStream(start int64) error {
	if _, err := file.VideoStream.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek in video stream: %w", err)
	}

	return nil
}

func (file *File) readFromVideoStream(bufferSize int64) ([]byte, int, error) {
	buffer := make([]byte, bufferSize)

	// Read data from the VideoStream (PartialReader) into the buffer
	n, err := file.VideoStream.Read(buffer[0:bufferSize])
	if err != nil {
		if err == io.EOF {
			return nil, 0, fmt.Errorf("end of file: %w", err)
		}

		return nil, 0, fmt.Errorf("failed to read from video stream: %w", err)
	}

	return buffer, n, nil
}

func (f *File) setResponseData(fileResponse *fuse.ReadResponse, buffer []byte, totalBytesRead int) {
	if totalBytesRead < len(buffer) {
		fmt.Printf("Read less than buffer size: %d/%d\n", totalBytesRead, len(buffer))
	}

	// Slice the buffer to the exact amount of bytes read
	fileResponse.Data = buffer[:totalBytesRead]
}
