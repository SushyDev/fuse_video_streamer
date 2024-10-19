package vlc

import (
	"fmt"
	"testing"
)

func TestWriteRead(t *testing.T) {
    buffer := NewBuffer(10, 0)
    testBuffer := make([]byte, 5)

    bytes := []byte{0, 1, 2, 3, 4}
    buffer.Write(bytes)
    // t.Logf("Buffer: %v", buffer.data)

    buffer.ReadAt(testBuffer, 0)
    // t.Logf("Test buffer: %v", testBuffer)

    for i, b := range testBuffer {
        if b != bytes[i] {
            t.Errorf("Expected %d, got %d", bytes[i], b)
        }
    }

    t.Log("Segment 1 -- Complete")

    bytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
    buffer.Write(bytes[5:])
    // t.Logf("Buffer: %v", buffer.data)

    buffer.ReadAt(testBuffer, 5)
    // t.Logf("Test buffer: %v", testBuffer)

    for i, b := range testBuffer {
        i += 5

        if b != bytes[i] {
            t.Errorf("Expected %d, got %d", bytes[i], b)
        }
    }

    t.Log("Segment 2 -- Complete")

    bytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}
    buffer.Write(bytes[10:])
    // t.Logf("Buffer: %v", buffer.data)

    buffer.ReadAt(testBuffer, 10)
    // t.Logf("Test buffer: %v", testBuffer)

    for i, b := range testBuffer {
        i += 10

        if b != bytes[i] {
            t.Errorf("Expected %d, got %d", bytes[i], b)
        }
    }

    t.Log("Segment 3 -- Complete")
}

func TestReadWriteWrap(t *testing.T) {
    buffer := NewBuffer(10, 0)
    var bytes []byte
    var testBuffer []byte
    var expectedWrittenData []byte
    var expectedReadData []byte

    bytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
    buffer.Write(bytes)
    // t.Logf("Buffer: %v", buffer.data)

    testBuffer = make([]byte, 5)
    buffer.ReadAt(testBuffer, 0)
    // t.Logf("Test buffer: %v", testBuffer)

    expectedReadData = []byte{0, 1, 2, 3, 4}

    for i, b := range testBuffer {
        if b != expectedReadData[i] {
            t.Errorf("Expected %d, got %d", expectedReadData[i], b)
        }
    }

    t.Log("Segment 1 -- Complete")

    bytes = []byte{10, 11, 12, 13, 14}
    buffer.Write(bytes)
    // t.Logf("Buffer: %v", buffer.data)

    expectedWrittenData = []byte{10, 11, 12, 13, 14, 5, 6, 7, 8, 9}

    for i, b := range buffer.data {
        if b != expectedWrittenData[i] {
            t.Errorf("Write: Expected %d, got %d", expectedWrittenData[i], b)
        }
    }


    testBuffer = make([]byte, 10)
    buffer.ReadAt(testBuffer, 5)
    // t.Logf("Test buffer: %v", testBuffer)

    expectedReadData = []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

    for i, b := range testBuffer {
        if b != expectedReadData[i] {
            t.Errorf("Read: Expected %d, got %d", expectedReadData[i], b)
        }
    }

    t.Log("Segment 2 -- Complete")
}




func TestIsPositionInBuffer(t *testing.T) {

    buffer := NewBuffer(10, 0)

    fmt.Println()
    val := buffer.IsPositionInBuffer(0)

    if val {
        t.Errorf("Expected false, got %v", val)
    }

    buffer.Write([]byte{1})

    fmt.Println()
    val = buffer.IsPositionInBuffer(0)

    if !val {
        t.Errorf("Expected true, got %v", val)
    }

    fmt.Println()
    val = buffer.IsPositionInBuffer(1)

    if val {
        t.Errorf("Expected false, got %v", val)
    }

    buffer.Write([]byte{2, 3, 4, 5, 6, 7, 8, 9, 10})

    fmt.Println()
    val = buffer.IsPositionInBuffer(0)

    if !val {
        t.Errorf("Expected true, got %v", val)
    }

    fmt.Println()
    val = buffer.IsPositionInBuffer(1)

    if !val {
        t.Errorf("Expected true, got %v", val)
    }

    fmt.Println()
    val = buffer.IsPositionInBuffer(10)

    if val {
        t.Errorf("Expected false, got %v", val)
    }

    fmt.Println()
    val = buffer.IsPositionInBuffer(11)

    if val {
        t.Errorf("Expected false, got %v", val)
    }

    buffer.ReadAt(make([]byte, 9), 0)

    buffer.Write([]byte{11, 12, 13, 14, 15})

    fmt.Println()
    val = buffer.IsPositionInBuffer(11)

    if !val {
        t.Errorf("Expected true, got %v", val)
    } 

    fmt.Println()
    val = buffer.IsPositionInBuffer(0)

    if val {
        t.Errorf("Expected false, got %v", val)
    } 

    fmt.Println()
    val = buffer.IsPositionInBuffer(1)

    if val {
        t.Errorf("Expected false, got %v", val)
    } 

    fmt.Println()
    val = buffer.IsPositionInBuffer(14)

    if !val {
        t.Errorf("Expected true, got %v", val)
    } 

    fmt.Println()
    val = buffer.IsPositionInBuffer(15)

    if val {
        t.Errorf("Expected false, got %v", val)
    } 
}

func TestGetBytesToOverwrite(t *testing.T) {
    buffer := NewBuffer(10, 0)

    // Initial state: empty buffer
    bytesToOverwrite := buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 10 {
        t.Errorf("Expected 10, got %d", bytesToOverwrite)
    }

    // Single Write
    buffer.Write([]byte{1, 2, 3, 4, 5})
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 5 {
        t.Errorf("Expected 5 after writing 5 bytes, got %d", bytesToOverwrite)
    }

    // Single Read
    buffer.ReadAt(make([]byte, 2), 0) // Reading 2 bytes
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 7 {
        t.Errorf("Expected 7 after reading 2 bytes, got %d", bytesToOverwrite)
    }

    // Overwriting: Write 3 more bytes
    buffer.Write([]byte{6, 7, 8})
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 4 {
        t.Errorf("Expected 4 after writing 3 more bytes, got %d", bytesToOverwrite)
    }

    // Write until the buffer is full
    buffer.Write([]byte{9, 10, 1, 2}) // This will fill the buffer completely
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 0 {
        t.Errorf("Expected 0 after filling the buffer, got %d", bytesToOverwrite)
    }

    // Read 5 bytes from the buffer w12 - r7
    buffer.ReadAt(make([]byte, 5), 2)
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 5 {
        t.Errorf("Expected 5 after reading 5 bytes, got %d", bytesToOverwrite)
    }

    // Write 3 more bytes
    buffer.Write([]byte{11, 12, 13})
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 2 {
        t.Errorf("Expected 2 after writing 3 bytes, got %d", bytesToOverwrite)
    }

    // Read 2 bytes
    buffer.ReadAt(make([]byte, 2), 7)
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 5 {
        t.Errorf("Expected 5 after reading 2 bytes, got %d", bytesToOverwrite)
    }

    // Write to wrap around (write more than bufferCap)
    buffer.Write([]byte{14, 15, 16, 17, 18, 19}) // Attempt to fill the buffer
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 0 {
        t.Errorf("Expected 0 after writing bytes beyond capacity, got %d", bytesToOverwrite)
    }

    // Read all bytes to empty the buffer
    buffer.ReadAt(make([]byte, 10), 0) // This will consume all remaining bytes
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 10 {
        t.Errorf("Expected 10 after emptying the buffer, got %d", bytesToOverwrite)
    }

    // Edge case: Try writing when buffer is empty
    buffer.Write([]byte{20, 21}) // Fill the buffer again
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 8 {
        t.Errorf("Expected 8 after writing 2 bytes, got %d", bytesToOverwrite)
    }

    // Final state: read all remaining bytes
    buffer.ReadAt(make([]byte, 8), 0) // Read remaining bytes
    bytesToOverwrite = buffer.GetBytesToOverwrite()
    if bytesToOverwrite != 10 {
        t.Errorf("Expected 10 after reading remaining bytes, got %d", bytesToOverwrite)
    }
}

