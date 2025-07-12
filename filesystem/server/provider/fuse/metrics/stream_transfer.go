package metrics

import (
	"sync/atomic"

	"github.com/google/uuid"
)

type StreamTransferMetrics struct {
	uuid string // Unique identifier for the stream transfer metrics

	streamId   string
	streamUrl  string
	streamSize int64

	transferOperations atomic.Int64
	transferBytes      atomic.Int64
	transferErrors     atomic.Int64

	finished atomic.Bool
}

type streamTransferMetricsJson struct {
	UUID               string `json:"uid"`
	StreamId           string `json:"stream_id"`
	StreamUrl          string `json:"stream_url"`
	StreamSize         int64  `json:"stream_size"`
	TransferOperations int64  `json:"transfer_operations"`
	TransferBytes      int64  `json:"transfer_bytes"`
	TransferErrors     int64  `json:"transfer_errors"`
	Finished           bool   `json:"finished"`
}

func (service *MetricsCollection) NewStreamTransferMetrics(streamId string, streamUrl string, streamSize int64) *StreamTransferMetrics {
	metrics := &StreamTransferMetrics{
		uuid:       uuid.New().String(),
		streamId:   streamId,
		streamUrl:  streamUrl,
		streamSize: streamSize,
	}

	if service.streamTransfers == nil {
		service.streamTransfers = make(map[uint64]*StreamTransferMetrics)
	}

	service.streamTransfers[uint64(len(service.streamTransfers)+1)] = metrics

	return metrics
}

func (metrics *StreamTransferMetrics) RecordTransferOperation(bytes int64, isError bool) {
	metrics.transferOperations.Add(1)
	metrics.transferBytes.Add(bytes)

	if isError {
		metrics.transferErrors.Add(1)
	}
}

func (metrics *StreamTransferMetrics) Finish() {
	metrics.finished.Store(true)
}

func (metrics *StreamTransferMetrics) ToJson() *streamTransferMetricsJson {
	return &streamTransferMetricsJson{
		UUID:               metrics.uuid,
		StreamId:           metrics.streamId,
		StreamUrl:          metrics.streamUrl,
		StreamSize:         metrics.streamSize,
		TransferOperations: metrics.transferOperations.Load(),
		TransferBytes:      metrics.transferBytes.Load(),
		TransferErrors:     metrics.transferErrors.Load(),
		Finished:           metrics.finished.Load(),
	}
}
