package vfs

import (
	"debrid_drive/real_debrid"
	"debrid_drive/stream"
	"fmt"
	"sync"
)

type File struct {
	ID       uint64
	Name     string
	VideoUrl string
	FetchUrl string
	Size     uint64

	VideoStreams sync.Map // map[uint64]*stream.Stream // map of video streams per PID

	fileSystem *FileSystem
}

func NewFile(fileSystem *FileSystem, parent *Directory, name string, videoUrl string, fetchUrl string, size uint64) (*File, error) {
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

		fileSystem: fileSystem,
	}

	return file, nil
}

func (file *File) GetVideoStream(pid uint32) (*stream.Stream, error) {
	existingVideoStream, ok := file.VideoStreams.Load(pid)
	if ok {
		return existingVideoStream.(*stream.Stream), nil
	}

	unrestrictedFile, err := real_debrid.UnrestrictLink(file.VideoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to unrestrict link: %w", err)
	}

	videoStream := stream.NewStream(unrestrictedFile.Download, file.Size)

	file.VideoStreams.Store(pid, videoStream)

	fmt.Printf("Created new video stream for PID %d\n", pid)

	return videoStream, nil
}
