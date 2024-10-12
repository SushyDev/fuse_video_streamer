package stream

import (
	"io"
)

func (pr *PartialReader) storeAsChunkInCache(startOffset int64, data []byte) cacheChunk {
	chunk := pr.getChunkByOffset(startOffset)

	pr.cacheMu.Lock()
	pr.cache.Add(chunk.number, data)
	pr.cacheMu.Unlock()

	return chunk
}

func (pr *PartialReader) getFromCache(chunkNumber int64) ([]byte, bool) {
	pr.cacheMu.Lock()
	data, ok := pr.cache.Get(chunkNumber)
	pr.cacheMu.Unlock()

	if !ok {
		return nil, false
	}

	return data.([]byte), true
}

func (pr *PartialReader) readFromCache(buffer []byte, bufferSize int64, chunk cacheChunk, chunkData []byte) (int, error) {
	start, end := getRelativeRangeInChunk(bufferSize, chunk, pr.offset)

	if start >= int64(len(chunkData)) {
		return 0, io.EOF
	}

	if end > int64(len(chunkData)) {
		end = int64(len(chunkData))
	}

	requestedBytes := chunkData[start:end]
	copySize := copy(buffer, requestedBytes)

	return copySize, nil
}
