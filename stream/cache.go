package stream

import (
	"io"
)

func storeAsChunkInCache(pr *PartialReader, startOffset int64, data []byte) cacheChunk {
	chunk := getChunkByStartOffset(startOffset, pr.Size)

	pr.cache.Add(chunk.number, data)

	return chunk
}

func (pr *PartialReader) readFromCache(p []byte, requestedReadSize int64, chunk cacheChunk, chunkData []byte) (int, error) {
	start, end := getRelativeRangeInChunk(requestedReadSize, chunk, pr.offset)

	if start >= int64(len(chunkData)) {
		return 0, io.EOF
	}

	if end > int64(len(chunkData)) {
		end = int64(len(chunkData))
	}

	requestedBytes := chunkData[start:end]
	copySize := copy(p, requestedBytes)

	return copySize, nil
}
