package vlc

import (
	"fmt"
	"sync"
	"sync/atomic"

	"debrid_drive/chart"
)

// FIFO buffer
type Buffer struct {
	data []byte
	mu   sync.Mutex

	startPosition atomic.Int64
	maxSize       int64

	chart *chart.Chart

	closed atomic.Bool
}

func NewBuffer(start int64, size int64, chart *chart.Chart) *Buffer {
	buffer := &Buffer{
		maxSize: size,
		data:    make([]byte, 0, size),
		chart:   chart,
	}

	buffer.SetStartPosition(start)

	return buffer
}

func (buffer *Buffer) UpdateDataChannel(position int64) {
	if buffer.IsClosed() {
		return
	}

	bufferStartPosition := buffer.GetStartPosition()
	bufferLen := bufferStartPosition + buffer.Len()

	select {
	case buffer.chart.ChartDataChannel <- chart.LinechartData{
		BufferCap:           bufferStartPosition,
		BufferLen:           bufferLen,
		SeekPosition:        position,
		BufferStartPosition: bufferStartPosition,
	}:
	default:
	}
}

func (buffer *Buffer) Write(p []byte) int {
	requestedSize := int64(len(p))

	if requestedSize <= 0 {
		return 0
	}

	if requestedSize > buffer.maxSize {
		buffer.chart.LogBuffer(fmt.Sprintf("Buffer is too small %d-%d\n", requestedSize, buffer.maxSize))
		return 0
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferSize := int64(len(buffer.data))
	bufferMaxSize := int64(cap(buffer.data))
	spaceLeft := bufferMaxSize - bufferSize
	overflow := requestedSize - spaceLeft

	if overflow > 0 && bufferSize > overflow {
		copy(buffer.data, buffer.data[overflow:])
		copy(buffer.data[bufferSize-overflow:], p)

		buffer.startPosition.Add(overflow)
	} else {
		buffer.data = append(buffer.data, p...)
	}

	// buffer.chart.LogBuffer(fmt.Sprintf("Buffered %d bytes\n", requestedSize))

	return len(p)
}

func (buffer *Buffer) Len() int64 {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	return int64(len(buffer.data))
}

func (buffer *Buffer) Cap() int64 {
	return int64(cap(buffer.data))
}

func (buffer *Buffer) ReadAt(p []byte, position int64) (int, error) {
	buffer.UpdateDataChannel(position)

	relativePosition := buffer.GetRelativePosition(position)

	if !buffer.IsPositionInBuffer(position) {
		buffer.chart.LogBuffer(fmt.Sprintf("Buffer is too small for seek %d-%d\n", position, relativePosition))
		return 0, fmt.Errorf("Buffer is too small for seek")
	}

	if !buffer.IsPositionInBuffer(position + int64(len(p))) {
		buffer.chart.LogBuffer(fmt.Sprintf("Buffer is too small for data %d-%d\n", position, relativePosition))
		return 0, fmt.Errorf("Buffer is too small for data")
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bytesRead := copy(p, buffer.data[relativePosition:])

	return bytesRead, nil
}

func (buffer *Buffer) Reset() {
	if buffer.IsClosed() {
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.data = make([]byte, 0, buffer.maxSize)
}

func (buffer *Buffer) Close() {
	if buffer.IsClosed() {
		return
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.data = make([]byte, 0, buffer.maxSize)
	buffer.data = nil

	buffer.closed.Store(true)
}

func (buffer *Buffer) IsClosed() bool {
	return buffer.closed.Load()
}

func (buffer *Buffer) GetStartPosition() int64 {
	return buffer.startPosition.Load()
}

func (buffer *Buffer) SetStartPosition(position int64) {
	buffer.chart.LogBuffer(fmt.Sprintf("Setting start position %d\n", position))
	buffer.startPosition.Store(position)
}

// Get relative position to buffer start
func (buffer *Buffer) GetRelativePosition(position int64) int64 {
	return position - buffer.GetStartPosition()
}

func (buffer *Buffer) OverflowByPosition(position int64) int64 {
	return buffer.GetRelativePosition(position) - buffer.Len()
}

func (buffer *Buffer) IsPositionInBuffer(position int64) bool {
	bufferSize := buffer.Len()

	if bufferSize <= 0 {
		return false
	}

	relativePosition := buffer.GetRelativePosition(position)

	if relativePosition < 0 {
		return false
	}

	if relativePosition > bufferSize {
		return false
	}

	return true
}

func (buffer *Buffer) WaitForPositionInBuffer(position int64) {
	for {
		if buffer.IsClosed() {
			return
		}

		if buffer.IsPositionInBuffer(position) {
			return
		}
	}
}

func (buffer *Buffer) GetBytesRead(position int64) int64 {
	relativePosition := buffer.GetRelativePosition(position)

	if relativePosition < 0 {
		return 0
	}

	return relativePosition
}

func (buffer *Buffer) GetBytesToOverwrite(position int64) int64 {
	// buffer.mu.Lock()
	// defer buffer.mu.Unlock()

	bufferSize := int64(len(buffer.data))
	bufferCap := int64(cap(buffer.data))
	bytesRead := buffer.GetBytesRead(position)

	var returnVal int64
	if bufferSize < buffer.maxSize {
		returnVal = min(max(0, bufferCap-bufferSize, bytesRead), bufferCap)
	} else {
		returnVal = buffer.GetBytesRead(position)
	}

	buffer.chart.LogBuffer(fmt.Sprintf("Bytes read/Buffer size (%d/%d) at position %d | Returning %d\n", bytesRead, bufferSize, position, returnVal))

	return returnVal
}
