package vfs

import (
	"debrid_drive/stream"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type File struct {
	identifier uint64
	name       string
	videoUrl   string
	fetchUrl   string
	size       uint64
	parent     *Directory

	videoStreams sync.Map // map[uint64]*stream.Stream // map of video streams per PID

	// mu sync.RWMutex TODO
}

func (file *File) GetIdentifier() uint64 {
	return file.identifier
}

func (file *File) GetName() string {
	return file.name
}

func (file *File) GetParent() *Directory {
	return file.parent
}

func (file *File) GetSize() uint64 {
	return file.size
}

func (file *File) SetSize(size uint64) {
	file.size = size
}

func (file *File) GetFetchUrl() string {
	return file.fetchUrl
}

func (file *File) SetFetchUrl(fetchUrl string) {
	file.fetchUrl = fetchUrl
}

func (file *File) GetVideoUrl() string {
	return file.videoUrl
}

func (file *File) SetVideoUrl(videoUrl string) {
	file.videoUrl = videoUrl
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
	file.name = name
}

func (file *File) Close() {
	file.videoStreams.Range(func(key, value interface{}) bool {
		stream := value.(*stream.Stream)
		stream.Close()

		return true
	})

	file.videoStreams.Clear()
}

func (file *File) getVideoStream(pid uint32) (*stream.Stream, error) {
	existingVideoStream, ok := file.videoStreams.Load(pid)
	if ok {
		return existingVideoStream.(*stream.Stream), nil
	}

	if file.videoUrl == "" && file.fetchUrl != "" {
		videoUrl, err := fetchVideoUrl(file.fetchUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch video URL: %w", err)
		}

		file.videoUrl = videoUrl
	}

	videoStream := stream.NewStream(file.videoUrl, file.size)

	file.videoStreams.Store(pid, videoStream)

	fmt.Printf("Created new video stream for PID %d\n", pid)

	return videoStream, nil
}

// Fetch video URL from the fetch URL
// TODO Fetch video URL AND video size
func fetchVideoUrl(fetchUrl string) (string, error) {
	response, err := http.Get(fetchUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch video URL: %w", err)
	}

	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(data), nil
}
