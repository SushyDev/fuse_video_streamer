package file

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
    "flag"

	"bazil.org/fuse"

	"debrid_drive/stream"
)

type File struct {
	VideoStream    *stream.PartialReader // Holds the open connection stream
	mu             sync.Mutex            // Mutex for thread safety
}

// Attr is called to retrieve the attributes (metadata) of the file.
func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	videoUrl := flag.Arg(1)

	// Initialize the VideoStream if it has not been done yet
	if f.VideoStream == nil {
		var err error
		f.VideoStream, err = stream.NewPartialReader(videoUrl) // Initialize the video stream
		if err != nil {
			return fmt.Errorf("failed to initialize video stream: %w", err)
		}
	}

	// Set the attributes
	a.Mode = 0444                                       // Read-only file
	a.Size = uint64(f.VideoStream.Size)                 // Set the size of the file
	// a.Blocks = (uint64(f.VideoStream.Size) + 511) / 512 // File system blocks
	a.Atime = time.Now()                                // Access time
	a.Mtime = time.Now()                                // Modification time
	a.Ctime = time.Now()                                // Creation time
	a.Uid = 1000                                        // User ID (change as needed)
	a.Gid = 1000                                        // Group ID (change as needed)

	return nil
}

// Read serves a file read request by buffering the stream and returning the requested data.
// This function reads data in chunks and supports seeking based on the request's Offset.
// Read calculates the read boundaries, seeks the video stream, reads the requested data, and sets the response.
func (file *File) Read(ctx context.Context, readRequest *fuse.ReadRequest, readReponse *fuse.ReadResponse) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	// Step 1: Calculate read boundaries
	start, bufferSize, err := file.calculateReadBoundaries(readRequest.Offset, int64(readRequest.Size))
	if err != nil {
		return err
	}

	// Step 2: Seek to the starting offset
	if err := file.seekVideoStream(start)
    err != nil {
		return err
	}

	// Step 3: Read data from the video stream
	buffer, totalBytesRead, err := file.readFromVideoStream(bufferSize)
	if err != nil {
		return err
	}

	// Step 4: Set response data
	file.setResponseData(readReponse, buffer, totalBytesRead)

	return nil
}

// calculateReadBoundaries checks and adjusts the start and buffer size for reading.
func (file *File) calculateReadBoundaries(start, requestedSize int64) (int64, int64, error) {
	fileSize := file.VideoStream.Size

	// Ensure the start position is within the bounds of the file
	if start >= fileSize {
		return 0, 0, fmt.Errorf("read position is beyond the end of the file")
	}

	// Ensure we don't read beyond the end of the file
	if start+requestedSize > fileSize {
		requestedSize = fileSize - start
	}

	return start, requestedSize, nil
}

// seekVideoStream sets the file position to the correct offset in the video stream.
func (file *File) seekVideoStream(start int64) error {
	if _, err := file.VideoStream.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek in video stream: %w", err)
	}

	return nil
}

// readFromVideoStream reads data from the video stream in chunks, leveraging caching if applicable.
func (file *File) readFromVideoStream(bufferSize int64) ([]byte, int, error) {
	buffer := make([]byte, bufferSize)

    // Read data from the VideoStream (PartialReader) into the buffer
    n, err := file.VideoStream.Read(buffer[0 : bufferSize])
    if err != nil {
        if err == io.EOF {
            return nil, 0, fmt.Errorf("end of file: %w", err)
        }

        return nil, 0, fmt.Errorf("failed to read from video stream: %w", err)
    }

	return buffer, n, nil
}

// setResponseData slices the buffer and assigns the data to the response.
func (f *File) setResponseData(fileResponse *fuse.ReadResponse, buffer []byte, totalBytesRead int) {
    if totalBytesRead < len(buffer) {
        fmt.Printf("Read less than buffer size: %d/%d\n", totalBytesRead, len(buffer))
    }

	// Slice the buffer to the exact amount of bytes read
	fileResponse.Data = buffer[:totalBytesRead]
}
