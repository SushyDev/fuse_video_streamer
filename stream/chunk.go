package stream

import (
	"debrid_drive/config"
    "fmt"
)

type Chunk struct {
	number      int64
	startOffset int64
	endOffset   int64
}

// getChunkByNumber creates a Chunk based on its number.
func getChunkByNumber(chunkNumber, fileSize int64) Chunk {
	startOffset := chunkNumber * config.CacheChunkSize
	endOffset := startOffset + config.CacheChunkSize

	if startOffset < 0 {
		startOffset = 0
	}

	if endOffset > fileSize {
		endOffset = fileSize
	}

	// endOffset = endOffset - 1

	return Chunk{
		number:      chunkNumber,
		startOffset: startOffset,
		endOffset:   endOffset,
	}
}

func getChunkByStartOffset(startOffset, fileSize int64) Chunk {
	chunkNumber := startOffset / config.CacheChunkSize

	if chunkNumber < 0 {
		chunkNumber = 0
	}

	return getChunkByNumber(chunkNumber, fileSize)
}

func getRelativeRangeInChunk(requestedReadSize int64, chunk Chunk, readerOffset int64) (int64, int64) {
	relativeOffset := readerOffset - chunk.startOffset
	if relativeOffset < 0 {
		relativeOffset = 0
	}


    start := relativeOffset
    end := relativeOffset + requestedReadSize

    if end > chunk.endOffset {
        end = chunk.endOffset
    }

    if end - start != requestedReadSize {
        fmt.Printf("Requested size is not equal to read size: %d/%d\n", end - start, requestedReadSize)
    }

	return start, end
}
