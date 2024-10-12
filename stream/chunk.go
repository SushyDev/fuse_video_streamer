package stream

import (
	"debrid_drive/config"
	"fmt"
)

type cacheChunk struct {
	number      int64
	startOffset int64
	endOffset   int64
}

func (pr *PartialReader) getChunkByNumber(chunkNumber int64) cacheChunk {
	fileSize := int64(pr.Size)
	startOffset := chunkNumber * config.CacheChunkSize
	endOffset := startOffset + config.CacheChunkSize

	// if startOffset < 0 {
	// 	startOffset = 0
	// }

	if endOffset > fileSize {
		endOffset = fileSize
	}

	return cacheChunk{
		number:      chunkNumber,
		startOffset: startOffset,
		endOffset:   endOffset,
	}
}

// Use startOffset
func (pr *PartialReader) getChunkByOffset(offset int64) cacheChunk {
	chunkNumber := offset / config.CacheChunkSize

	if chunkNumber < 0 {
		chunkNumber = 0
	}

	return pr.getChunkByNumber(chunkNumber)
}

func getRelativeRangeInChunk(bufferSize int64, chunk cacheChunk, readerOffset int64) (int64, int64) {
	relativeOffset := readerOffset - chunk.startOffset
	if relativeOffset < 0 {
		relativeOffset = 0
	}

	start := relativeOffset
	end := relativeOffset + bufferSize

	if end > chunk.endOffset {
		end = chunk.endOffset
	}

	if end-start != bufferSize {
		fmt.Printf("Requested size is not equal to read size: %d/%d\n", end-start, bufferSize) // --- Investigate why this happens with the last chunk
	}

	return start, end
}
