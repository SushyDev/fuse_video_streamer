package vfs

import (
	"debrid_drive/stream"
	"fmt"
	"io"
	"sync"
)

type File struct {
	ID       uint64
	Name     string
	VideoUrl string
	FetchUrl string
	Size     uint64
	Parent   *Directory

	VideoStreams sync.Map // map[uint64]*stream.Stream // map of video streams per PID

	// mu sync.RWMutex TODO

	fileSystem *VirtualFileSystem
}

func NewFile(fileSystem *VirtualFileSystem, parent *Directory, name string, videoUrl string, fetchUrl string, size uint64) (*File, error) {
	if fileSystem == nil {
		return nil, fmt.Errorf("file system is nil")
	}

	if parent == nil {
		return nil, fmt.Errorf("parent directory is nil")
	}

	ID := fileSystem.IDCounter.Add(1)

	file := &File{
		ID:       ID,
		Name:     name,
		VideoUrl: videoUrl,
		FetchUrl: fetchUrl,
		Size:     size,
		Parent:   parent,

		fileSystem: fileSystem,
	}

	return file, nil
}

func (file *File) Read(p []byte, offset int64, pid uint32) (int, error) {
	videoStream, err := file.getVideoStream(pid)
	if err != nil {
		return 0, fmt.Errorf("failed to get video stream: %w", err)
	}

	_, err = videoStream.Seek(uint64(offset), io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("failed to seek in video stream: %w", err)
	}

	bytesRead, err := videoStream.Read(p)
	if err != nil {
		return 0, fmt.Errorf("failed to read from video stream: %w", err)
	}

	return bytesRead, nil
}

func (file *File) Rename(name string) {
	file.Name = name
}

func (file *File) Close() {
	file.VideoStreams.Range(func(key, value interface{}) bool {
		stream := value.(*stream.Stream)
		stream.Close()

		return true
	})

	file.VideoStreams.Clear()
}

func (file *File) getVideoStream(pid uint32) (*stream.Stream, error) {
	existingVideoStream, ok := file.VideoStreams.Load(pid)
	if ok {
		return existingVideoStream.(*stream.Stream), nil
	}

	if file.VideoUrl == "" && file.FetchUrl != "" {
		// todo fetch http
	}

    file.VideoUrl = ""

	videoStream := stream.NewStream(file.VideoUrl, file.Size)

	file.VideoStreams.Store(pid, videoStream)

	fmt.Printf("Created new video stream for PID %d\n", pid)

	return videoStream, nil
}
