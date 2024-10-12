package vfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"bazil.org/fuse"

	"debrid_drive/logger"
	"debrid_drive/real_debrid"
	"debrid_drive/stream"
)

type File struct {
	name        string
	iNode       uint64
	videoUrl    string                // URL of the video file
	videoStream *stream.PartialReader // Holds the open connection stream
	mu          sync.RWMutex
	chunks      int64
	size        int64
}

// Attr is called to retrieve the attributes (metadata) of the file.
func (file *File) Attr(ctx context.Context, a *fuse.Attr) error {
	logger.Logger.Infof("Getting attributes for file %s\n inode %d", file.name, file.iNode)

	a.Inode = file.iNode
	a.Mode = os.ModePerm
	a.Size = uint64(file.size)
	// a.Blocks = (uint64(f.VideoStream.Size) + 511) / 512 // File system blocks
	a.Atime = time.Unix(0, 0)
	a.Mtime = time.Unix(0, 0)
	a.Ctime = time.Unix(0, 0)

	return nil
}

func (file *File) Remove(ctx context.Context) error {
	file.mu.Lock()
	defer file.mu.Unlock()

	videoStream, err := file.getVideoStream()
	if err != nil {
		return err
	}

	return videoStream.Close()
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

	file.populateReadResponse(readReponse, buffer, totalBytesRead)

	return nil
}

func (file *File) getVideoStream() (*stream.PartialReader, error) {
	if file.videoStream != nil {
		return file.videoStream, nil
	}

	var err error

	unrestrictedFile, err := real_debrid.UnrestrictLink(file.videoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to unrestrict link: %w", err)
	}

	file.videoStream, err = stream.NewPartialReader(unrestrictedFile.Download, file.size)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize video stream: %w", err)
	}

	logger.Logger.Infof("Initialized video stream for file %s\n", file.videoUrl)

	if file.videoStream.Size != file.size {
		logger.Logger.Error("Size mismatch between file and video stream")
	}

	return file.videoStream, nil
}

func (file *File) calculateReadBoundaries(start, requestedSize int64) (int64, int64, error) {
	if start >= file.size {
		return 0, 0, io.EOF
	}

	if start+requestedSize > file.size {
		requestedSize = file.size - start
	}

	return start, requestedSize, nil
}

func (file *File) seekVideoStream(start int64) error {
	videoStream, err := file.getVideoStream()
	if err != nil {
		return err
	}

	if _, err := videoStream.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek in video stream: %w", err)
	}

	return nil
}

func (file *File) readFromVideoStream(bufferSize int64) ([]byte, int, error) {
	videoStream, err := file.getVideoStream()
	if err != nil {
		return nil, 0, err
	}

	buffer := make([]byte, bufferSize)

	// Read data from the VideoStream (PartialReader) into the buffer
	n, err := videoStream.Read(buffer[0:bufferSize])
	if err != nil {
		if err == io.EOF {
			return nil, 0, fmt.Errorf("end of file: %w", err)
		}

		return nil, 0, fmt.Errorf("failed to read from video stream: %w", err)
	}

	return buffer, n, nil
}

func (file *File) populateReadResponse(fileResponse *fuse.ReadResponse, buffer []byte, totalBytesRead int) {
	if totalBytesRead < len(buffer) {
		logger.Logger.Infof("Read less than buffer size: %d/%d\n", totalBytesRead, len(buffer))
	}

	// Slice the buffer to the exact amount of bytes read
	fileResponse.Data = buffer[:totalBytesRead]
}
