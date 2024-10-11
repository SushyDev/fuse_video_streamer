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

func getChunkByNumber(chunkNumber, fileSize int64) cacheChunk {
	startOffset := chunkNumber * config.CacheChunkSize
	endOffset := startOffset + config.CacheChunkSize

	if startOffset < 0 {
		startOffset = 0
	}

	if endOffset > fileSize {
		endOffset = fileSize
	}

	return cacheChunk{
		number:      chunkNumber,
		startOffset: startOffset,
		endOffset:   endOffset,
	}
}

func getChunkByStartOffset(startOffset, fileSize int64) cacheChunk {
	chunkNumber := startOffset / config.CacheChunkSize

	if chunkNumber < 0 {
		chunkNumber = 0
	}

	return getChunkByNumber(chunkNumber, fileSize)
}

func getRelativeRangeInChunk(requestedReadSize int64, chunk cacheChunk, readerOffset int64) (int64, int64) {
	relativeOffset := readerOffset - chunk.startOffset
	if relativeOffset < 0 {
		relativeOffset = 0
	}

	start := relativeOffset
	end := relativeOffset + requestedReadSize

	if end > chunk.endOffset {
		end = chunk.endOffset
	}

	if end-start != requestedReadSize {
		fmt.Printf("Requested size is not equal to read size: %d/%d\n", end-start, requestedReadSize) // --- Investigate why this happens with the last chunk
	}

	return start, end
}
