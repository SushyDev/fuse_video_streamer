package stream

import (
	"fmt"
	"io"
)

func storeAsChunkInCache(pr *PartialReader, startOffset int64, data []byte) (Chunk) {
	chunk := getChunkByStartOffset(startOffset, pr.Size)

	pr.cache.Add(chunk.number, data)

    return chunk
}

// readFromCache reads the requested bytes from the cached chunk data.
func (pr *PartialReader) readFromCache(p []byte, requestedReadSize int64, chunk Chunk, chunkData []byte) (int, error) {
	start, end := getRelativeRangeInChunk(requestedReadSize, chunk, pr.offset)

	if start >= int64(len(chunkData)) {
		fmt.Printf("Start is beyond the end of the chunk: %d\n", start)
		return 0, io.EOF
	}

	if end > int64(len(chunkData)) {
		fmt.Printf("End is beyond the end of the chunk: %d\n", end)
		end = int64(len(chunkData))
	}

	requestedBytes := chunkData[start:end]
	copySize := copy(p, requestedBytes)

	return copySize, nil
}
