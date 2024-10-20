package buffer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Buffer struct {
	data          []byte
	startPosition atomic.Int64 // The logical start position of the buffer

	readPosition  atomic.Int64 // The position where the next read will happen
	writePosition atomic.Int64 // The position where the next write will happen

	count atomic.Int64 // The number of bytes currently in the buffer

	readPage  atomic.Int64
	writePage atomic.Int64

	mu sync.RWMutex

	closed bool
}

func NewBuffer(size int64, startPosition int64) *Buffer {
	buffer := &Buffer{
		data: make([]byte, size),
	}

	buffer.SetStartPosition(startPosition)

	return buffer
}

func (buffer *Buffer) Cap() int64 {
	return int64(cap(buffer.data))
}

func (buffer *Buffer) ReadAt(p []byte, position int64) (int, error) {
	if buffer.closed {
		return 0, errors.New("buffer is closed")
	}

	buffer.mu.RLock()
	defer buffer.mu.RUnlock()

	bufferCount := buffer.count.Load()
	if bufferCount <= 0 {
		return 0, errors.New("buffer is empty")
	}

	if !buffer.IsPositionInBuffer(position) {
		return 0, errors.New(fmt.Sprintf("position %d is not in buffer", position))
	}

	bufferCap := buffer.Cap()
	relativePos := buffer.GetRelativePosition(position)
	bufferPosition := relativePos % bufferCap

	readPosition := buffer.readPosition.Load()

	writePosition := buffer.writePosition.Load()

	requestedSize := int64(len(p))

	// fmt.Println("ReadAt: bufferPos", bufferPos, "readPosition", readPosition, "writePosition", writePosition, "bufferCount", bufferCount, "requestedSize", requestedSize, "relativePos", relativePos)

	var readSize int64
	if bufferCount == bufferCap && readPosition == writePosition {
		readSize = min(requestedSize, bufferCap)
	} else if writePosition >= bufferPosition {
		readSize = min(requestedSize, writePosition-bufferPosition)
	} else {
		readSize = min(requestedSize, bufferCap-bufferPosition+writePosition)
	}

	if bufferPosition+readSize <= bufferCap {
		copy(p, buffer.data[bufferPosition:bufferPosition+readSize])
	} else {
		firstPart := bufferCap - bufferPosition
		copy(p, buffer.data[bufferPosition:bufferCap])
		copy(p[firstPart:], buffer.data[0:readSize-firstPart])
	}

	newReadPosition := (bufferPosition + readSize) % bufferCap
	if newReadPosition <= readPosition {
		buffer.readPage.Add(1)
	}

	buffer.readPosition.Store(newReadPosition)

	buffer.count.Store(bufferCount - readSize)

	// fmt.Printf("Read position %d, Read page %d, n %d\n", newReadPosition, readPage, n)

	return int(readSize), nil
}

// Write writes data to the ring buffer from p.
func (buffer *Buffer) Write(p []byte) (int, error) {
	if buffer.closed {
		return 0, errors.New("buffer is closed")
	}

	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()
	requestedSize := int64(len(p))

	if requestedSize > bufferCap {
		return 0, fmt.Errorf("write data exceeds buffer size: %d", requestedSize)
	}

	availableSpace := buffer.GetBytesToOverwrite()
	if requestedSize > availableSpace {
		return 0, fmt.Errorf("not enough space in buffer: %d/%d", requestedSize, availableSpace)
	}

	bufferCount := buffer.count.Load()
	writePosition := buffer.writePosition.Load()

	// if buffer is not full yet we need to append rather than copy
	if writePosition+requestedSize <= bufferCap {
		// No wraparound needed.
		copy(buffer.data[writePosition:], p)
	} else {
		firstPart := bufferCap - writePosition
		copy(buffer.data[writePosition:], p[:firstPart])
		copy(buffer.data[0:], p[firstPart:])
	}

	newWritePosition := (writePosition + requestedSize) % bufferCap
	if newWritePosition <= writePosition {
		buffer.writePage.Add(1)
	}

	buffer.writePosition.Store(newWritePosition)

	buffer.count.Store(bufferCount + requestedSize)

	// fmt.Printf("Write position %d, Write page %d\n", newWritePosition, writePage)

	return int(requestedSize), nil
}

func (buffer *Buffer) SetStartPosition(position int64) {
	buffer.startPosition.Store(position)
}

func (buffer *Buffer) GetStartPosition() int64 {
	return buffer.startPosition.Load()
}

func (buffer *Buffer) GetRelativePosition(position int64) int64 {
	return position - buffer.startPosition.Load()
}

// Checks if the given logical position is within the readPos and writePos.
func (buffer *Buffer) IsPositionInBufferSync(position int64) bool {
	buffer.mu.RLock()
	defer buffer.mu.RUnlock()

	return buffer.IsPositionInBuffer(position)
}

func (buffer *Buffer) IsPositionInBuffer(position int64) bool {
	relativePosition := buffer.GetRelativePosition(position)
	if relativePosition < 0 {
		return false
	}

	bufferCap := buffer.Cap()
	bufferPosition := relativePosition % bufferCap
	bufferPositionPage := relativePosition / bufferCap

	readPage := buffer.readPage.Load()
	readPosition := buffer.readPosition.Load()
	writePage := buffer.writePage.Load()
	writePosition := buffer.writePosition.Load()

    // Case 0: The buffer is empty.
    if readPage == 0 && writePage == 0 && readPosition == 0 && writePosition == 0 {
        return false
    }

	if readPage == writePage {
		// Case 1: Same page, position must be between readPosition and writePosition.
		return bufferPosition >= readPosition && bufferPosition < writePosition
	}

	if bufferPositionPage == readPage {
		// Case 2: Position is on the read page.
		return bufferPosition >= readPosition
	}

	if bufferPositionPage == writePage {
		// Case 3: Position is on the write page.
		return bufferPosition < writePosition
	}

	// Case 4: Position is in between readPage and writePage when they are not the same.
	return readPage < writePage
}

func (buffer *Buffer) WaitForPositionInBuffer(position int64, context context.Context) {
	for {
		if buffer.closed {
			return
		}

		if buffer.IsPositionInBufferSync(position) {
			return
		}

		select {
		case <-context.Done():
			return
		case <-time.After(100 * time.Microsecond):
		}

	}
}

func (buffer *Buffer) GetBytesToOverwriteSync() int64 {
	buffer.mu.RLock()
	defer buffer.mu.RUnlock()

	return buffer.GetBytesToOverwrite()
}

func (buffer *Buffer) GetBytesToOverwrite() int64 {
	if buffer.closed {
		return 0
	}

	bufferCap := buffer.Cap()
	bufferCount := buffer.count.Load()

	return bufferCap - bufferCount
}

func (buffer *Buffer) Reset(position int64) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.SetStartPosition(position)
	buffer.writePosition.Store(0)
	buffer.readPosition.Store(0)
	buffer.count.Store(0)
	buffer.writePage.Store(0)
	buffer.readPage.Store(0)
	buffer.data = make([]byte, buffer.Cap())
}

func (buffer *Buffer) Close() {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.data = nil

	buffer.closed = true
}
