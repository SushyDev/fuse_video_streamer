package vlc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

type Buffer struct {
	data          []byte
	startPosition atomic.Int64 // The logical start position of the buffer

	readPosition  atomic.Int64 // The position where the next read will happen
	writePosition atomic.Int64 // The position where the next write will happen
	count         atomic.Int64 // The number of bytes currently in the buffer

	readPage  atomic.Int64
	writePage atomic.Int64

	mu sync.Mutex
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

func (buffer *Buffer) IsFull() bool {
	return buffer.count.Load() == buffer.Cap()
}

func (buffer *Buffer) IsEmpty() bool {
	return buffer.count.Load() == 0
}

func (buffer *Buffer) ReadAt(p []byte, position int64) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()
	relativePos := buffer.GetRelativePosition(position)
	bufferPos := relativePos % bufferCap

	if !buffer.IsPositionInBuffer(position) {
		return 0, errors.New(fmt.Sprintf("position %d is not in buffer", position))
	}

	readPosition := buffer.readPosition.Load()
	readPage := buffer.readPage.Load()

	writePosition := buffer.writePosition.Load()

	bufferCount := buffer.count.Load()
	requestedSize := int64(len(p))

	if bufferCount <= 0 {
		return 0, errors.New("buffer is empty")
	}

	// fmt.Println("ReadAt: bufferPos", bufferPos, "readPosition", readPosition, "writePosition", writePosition, "bufferCount", bufferCount, "requestedSize", requestedSize, "relativePos", relativePos)

	var n int64
	if bufferCount == bufferCap && readPosition == writePosition {
		n = min(requestedSize, bufferCap)
	} else if writePosition >= bufferPos {
		n = min(requestedSize, writePosition-bufferPos)
	} else {
		n = min(requestedSize, bufferCap-bufferPos+writePosition)
	}

	if bufferPos+n <= bufferCap {
		copy(p, buffer.data[bufferPos:bufferPos+n])
	} else {
		firstPart := bufferCap - bufferPos
		copy(p, buffer.data[bufferPos:bufferCap])
		copy(p[firstPart:], buffer.data[0:n-firstPart])
	}

	newReadPosition := (bufferPos + n) % bufferCap
	if newReadPosition <= readPosition {
		readPage++
	}

	buffer.readPosition.Store(newReadPosition)
	buffer.readPage.Store(readPage)

	buffer.count.Store(bufferCount - n)

	// fmt.Printf("Read position %d, Read page %d, n %d\n", newReadPosition, readPage, n)

	return int(n), nil
}

// Write writes data to the ring buffer from p.
func (buffer *Buffer) Write(p []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()
	requestedSize := int64(len(p))

	if requestedSize > bufferCap {
		return 0, errors.New(fmt.Sprintf("write data exceeds buffer size: %d", requestedSize))
	}

	bufferCount := buffer.count.Load()

	availableSpace := bufferCap - bufferCount
	if requestedSize > availableSpace {
		return 0, errors.New(fmt.Sprintf("not enough space in buffer: %d", availableSpace))
	}

	writePosition := buffer.writePosition.Load()
	writePage := buffer.writePage.Load()

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
		writePage++
	}

	buffer.writePosition.Store(newWritePosition)
	buffer.writePage.Store(writePage)

	buffer.count.Store(bufferCount + requestedSize)

	// fmt.Printf("Write position %d, Write page %d\n", newWritePosition, writePage)

	return int(requestedSize), nil
}

// OverflowByPosition checks how much the given logical position exceeds the writePos.
// It returns a positive overflow value if the position exceeds the writePos,
// or zero if the position is within or behind the writePos.
func (buffer *Buffer) OverflowByPosition(position int64) int64 {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	relativePosition := buffer.GetRelativePosition(position)
	if relativePosition < 0 {
		panic("position is behind the buffer start position")
	}

	// Calculate the buffer position using the modulo operation.
	bufferPos := relativePosition % bufferCap
	writePosition := buffer.writePosition.Load()

	if bufferPos >= writePosition {
		return int64(bufferPos - writePosition)
	}

	return 0
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
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	return buffer.IsPositionInBuffer(position)
}

func (buffer *Buffer) IsPositionInBuffer(position int64) bool {
	relativePosition := buffer.GetRelativePosition(position)
	if relativePosition < 0 {
		return false
	}

	bufferCap := buffer.Cap()

	bufferPosition := relativePosition % bufferCap
	bufferPositionPage := relativePosition / buffer.Cap()

	readPage := buffer.readPage.Load()
	readPosition := buffer.readPosition.Load()

	writePosition := buffer.writePosition.Load()
	writePage := buffer.writePage.Load()

	// fmt.Printf("IsPositionInBuffer: position %d, bufferPosition %d, positionPage %d, readPosition %d, readPage %d, writePosition %d, writePage %d\n", position, bufferPosition, bufferPositionPage, readPosition, readPage, writePosition, writePage)

	if readPage == writePage {
		if bufferPosition >= readPosition && bufferPosition < writePosition {
			return true
		}

		return false
	}

	if readPage < writePage {
		if bufferPositionPage == readPage {
			// check is the position is bigger or equal to read position and less than write position of of next page
			writePositionNextPage := writePosition + bufferCap*(writePage-readPage)

			if bufferPosition >= readPosition && bufferPosition < writePositionNextPage {
				return true
			}

			return false
		}

		if bufferPositionPage == writePage {
			// check if the position is bigger or equal to read position of last page and less than write position
			readPositionLastPage := readPosition - bufferCap*(writePage-readPage)

			if bufferPosition >= readPositionLastPage && bufferPosition < writePosition {
				return true
			}

			return false
		}
	}

	if readPage > writePage {
		if bufferPositionPage == readPage {
			// check if the position is bigger or equal to read position and less than write position of last page
			writePositionLastPage := writePosition + bufferCap*(readPage-writePage)

			if bufferPosition >= readPosition && bufferPosition < writePositionLastPage {
				return true
			}

			return false
		}

		if bufferPositionPage == writePage {
			// check if the position is bigger or equal to read position of next page and less than write position
			readPositionNextPage := readPosition + bufferCap*(readPage-writePage)

			if bufferPosition >= readPositionNextPage && bufferPosition < writePosition {
				return true
			}

			return false
		}
	}

	return false
}

func (buffer *Buffer) WaitForPositionInBuffer(position int64, context context.Context) {
	for {
		if buffer.IsPositionInBufferSync(position) {
			return
		}

		select {
		case <-context.Done():
			return
        default:
		}

	}
}

func (buffer *Buffer) GetBytesToOverwrite() int64 {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()
	bufferCount := buffer.count.Load()

	readPosition := buffer.readPosition.Load()
	writePosition := buffer.writePosition.Load()

	// fmt.Printf("GetBytesToOverwrite: readPosition %d, writePosition %d, bufferCount %d\n", readPosition, writePosition, bufferCount)

	// Case 1: Buffer is empty
	if bufferCount == 0 {
		return bufferCap
	}

	// Case 2: Write position is ahead of read position
	if writePosition > readPosition {
		staleBytes := writePosition - readPosition

		return bufferCap - staleBytes
	}

	// Case 3: Write position has wrapped around to the start
	if writePosition < readPosition {
		staleBytes := bufferCap - readPosition + writePosition

		return bufferCap - staleBytes
	}

	// Should not reach here
	return 0
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

func (buffer *Buffer) Close() {}
