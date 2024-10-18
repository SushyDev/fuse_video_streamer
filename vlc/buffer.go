package vlc

import (
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

// // Read reads data from the ring buffer into p.
// func (buffer *Buffer) Read(p []byte) (int, error) {
//     buffer.mu.Lock()
//     defer buffer.mu.Unlock()
//
//     bufferCap := buffer.Cap()
//     readPosition := buffer.readPosition
//     writePosition := buffer.writePosition
//
//     // If the buffer is empty (readPos == writePos), there's nothing to read.
//     if buffer.readPosition == buffer.writePosition {
//         return 0, io.EOF
//     }
//
//     // Determine the number of bytes to read.
//     var n int
//     if writePosition > readPosition {
//         n = min(len(p), writePosition-readPosition)
//     } else {
//         // bufferLen ipv bufferCap
//         n = min(len(p), bufferCap-readPosition)
//     }
//
//     // Read the data into p.
//     copy(p, buffer.data[buffer.readPosition:buffer.readPosition+n])
//
//     // BufferLen ipv bufferCap
//     buffer.readPosition = (buffer.readPosition + n) % bufferCap
//
//     return n, nil
// }

// ReadAt reads data from the ring buffer at a specific logical position.
func (buffer *Buffer) ReadAt(p []byte, position int64) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	relativePos := buffer.GetRelativePosition(position)
	if relativePos < 0 || relativePos >= bufferCap {
        fmt.Printf("Position out of range: %d\n", relativePos)
		return 0, errors.New("position out of range")
	}

	bufferPos := relativePos % bufferCap
    readPosition := buffer.readPosition.Load()
	writePosition := buffer.writePosition.Load()

	requestedSize := int64(len(p))

	// Determine the number of bytes to read.
	var n int64
    if writePosition > bufferPos {
        n = min(requestedSize, writePosition-bufferPos)
    } else {
        n = min(requestedSize, bufferCap-bufferPos)
    }

	// Read the data into p.
	copy(p, buffer.data[bufferPos:bufferPos+n])
    buffer.readPosition.Store((readPosition + n) % bufferCap)

	return int(n), nil
}

// Write writes data to the ring buffer from p.
func (buffer *Buffer) Write(p []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	requestedSize := int64(len(p))
	if requestedSize > bufferCap {
        fmt.Printf("Write data exceeds buffer size: %d\n", requestedSize)
		return 0, errors.New("write data exceeds buffer size")
	}

	// // Calculate the available space between readPos and writePos.
	// availableSpace := (rb.size + rb.readPosition - rb.writePosition - 1) % rb.size
	//
	// if n > availableSpace {
	//     return 0, errors.New("not enough space in buffer")
	// }

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

    buffer.writePosition.Store((writePosition + requestedSize) % bufferCap)

	return int(requestedSize), nil
}

// OverflowByPosition checks how much the given logical position exceeds the writePos.
// It returns a positive overflow value if the position exceeds the writePos,
// or zero if the position is within or behind the writePos.
func (buffer *Buffer) OverflowByPosition(position int64) int64 {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	// Calculate the relative position in the buffer.
	relativePosition := buffer.GetRelativePosition(position)
	if relativePosition < 0 {
		// If the relative position is negative, it means the position is behind the startPosition.
		// This means the data has been overwritten, so the overflow is considered to be maximal.
		return int64(bufferCap)
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
func (buffer *Buffer) IsPositionInBuffer(position int64) bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	relativePosition := buffer.GetRelativePosition(position)
	if relativePosition < 0 {
		return false
	}

	bufferPos := relativePosition % bufferCap

	writePosition := buffer.writePosition.Load()
	readPosition := buffer.readPosition.Load()

    if readPosition == writePosition {
        return false
    }

    // fmt.Printf("Check position %d, bufferPos %d, readPos %d, writePos %d\n", position, bufferPos, readPosition, writePosition)

	if readPosition < writePosition {
		return bufferPos >= readPosition && bufferPos <= writePosition
	}


    return bufferPos >= readPosition || bufferPos <= writePosition

}

func (buffer *Buffer) WaitForPositionInBuffer(position int64) {
	for !buffer.IsPositionInBuffer(position) {
		time.Sleep(10 * time.Millisecond)
	}
}

func (buffer *Buffer) GetBytesToOverwrite(position int64) int64 {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	bufferCap := buffer.Cap()

	relativePosition := buffer.GetRelativePosition(position)
	// if relativePosition < 0 {
	//     return 0
	// }

	bufferPos := relativePosition % bufferCap

	writePosition := buffer.writePosition.Load()
	readPosition := buffer.readPosition.Load()

	if readPosition <= writePosition {
		if bufferPos >= writePosition {
			return int64(bufferCap - bufferPos + readPosition)
		}

		return int64(readPosition - bufferPos)
	}

	return int64(readPosition - bufferPos)
}

func (buffer *Buffer) Reset(position int64) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()

	buffer.SetStartPosition(position)
	buffer.writePosition.Store(0)
	buffer.readPosition.Store(0)
	buffer.data = make([]byte, buffer.Cap())
}

func (buffer *Buffer) Close() {}
