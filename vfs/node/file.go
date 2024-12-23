package node

import (
	"fmt"
	"fuse_video_steamer/stream"
	"io"
	"net/http"
	"sync"
)

type File struct {
	node *Node

	size uint64
	host string

	// Todo videoStream struct
	videoStreams sync.Map // map[uint64]*stream.Stream // map of video streams per PID

	// mu sync.RWMutex TODO
}

func NewFile(node *Node, size uint64, host string) *File {
	return &File{
		node: node,
		size: size,
		host: host,
	}
}

func (file *File) GetNode() *Node {
	return file.node
}

func (file *File) GetSize() uint64 {
	return file.size
}

func (file *File) SetSize(size uint64) {
	file.size = size
}

func (file *File) GetHost() string {
	return file.host
}

// func (file *File) Move(newParent *Directory, newName string) {
// 	file.parent = newParent
// 	file.name = newName
// }

func (file *File) Link(parent *Directory, name string) *File {
	return nil
}

func (file *File) GetVideoURL() string {
	return ""
	// return file.videoURL
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
	// file.name = name
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

	// if file.videoURL == "" && file.host != "" {
	// 	videoUrl, err := fetchVideoUrl(file.host)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to fetch video URL: %w", err)
	// 	}
	//
	// 	file.videoURL = videoUrl
	// }

	videoUrl := ""

	videoStream := stream.NewStream(videoUrl, file.size)

	file.videoStreams.Store(pid, videoStream)

	fmt.Printf("Created new video stream for PID %d\n", pid)

	return videoStream, nil
}

// Fetch video URL from the fetch URL
// TODO Fetch video URL AND video size
func fetchVideoUrl(host string) (string, error) {
	response, err := http.Get(host)
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
