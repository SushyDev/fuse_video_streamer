package stream

import (
	"debrid_drive/config"
	"debrid_drive/logger"
)

type cacheChunk struct {
	number int64
}

func (chunk *cacheChunk) getData(pr *PartialReader) (data []byte, err error) {
	data, ok := pr.cacheManager.getChunkDataFromCache(chunk.number)
	if ok {
		return data, nil
	}

	data, ok = pr.getChunkDataFromOngoingPrefetch(chunk.number)
	if ok {
		return data, nil
	}

	logger.Logger.Infof("Fetching chunk %d", chunk.number)

	data, error := pr.fetchAndCacheChunkData(chunk)
	if error != nil {
		return nil, error
	}

	return data, nil
}

func (chunk *cacheChunk) getRange() (start int64, end int64) {
	start = chunk.number * config.CacheChunkSize
	end = start + config.CacheChunkSize

	return start, end
}

func (chunk *cacheChunk) getSize() int64 {
	start, end := chunk.getRange()

	return end - start
}

func GetChunkByNumber(chunkNumber int64) *cacheChunk {
	return &cacheChunk{
		number: chunkNumber,
	}
}

func GetChunkByOffset(offset int64) *cacheChunk {
	chunkNumber := offset / config.CacheChunkSize

	if chunkNumber < 0 {
		chunkNumber = 0
	}

	return GetChunkByNumber(chunkNumber)
}
