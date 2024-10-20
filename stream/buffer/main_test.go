package buffer

import (
	"fmt"
	"testing"
)

const startPositionOffet = 349087234

func TestWrite(t *testing.T) {
	buffer := NewBuffer(10, startPositionOffet)
	buffer.Write([]byte{0, 1, 2, 3, 4})

	for i, b := range buffer.data {
		if i < 5 && b != byte(i) {
			t.Errorf("Expected %d, got %d", i, b)
		} else if i >= 5 && b != 0 {
			t.Errorf("Expected 0, got %d", b)
		}
	}

	writePosition := buffer.writePosition.Load()

	if writePosition != 5 {
		t.Errorf("Expected 5, got %d", writePosition)
	}

	buffer.Write([]byte{5, 6, 7, 8, 9})

	for i, b := range buffer.data {
		if i < 10 && b != byte(i) {
			t.Errorf("Expected %d, got %d", i, b)
		} else if i >= 10 && b != byte(i-10) {
			t.Errorf("Expected %d, got %d", i-10, b)
		}
	}

	if buffer.writePosition.Load() != 0 {
		t.Errorf("Expected 0, got %d", buffer.writePosition.Load())
	}

	_, err := buffer.ReadAt(make([]byte, 5), startPositionOffet+0)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

	buffer.Write([]byte{10, 11, 12, 13, 14})

	if buffer.writePosition.Load() != 5 {
		t.Errorf("Expected 5, got %d", buffer.writePosition.Load())
	}

	bytesToOverflow := []byte{15, 16, 17, 18, 19, 20, 21, 22, 23, 24}

	_, err = buffer.Write(bytesToOverflow)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	requestedSize := len(bytesToOverflow)

	if err.Error() != fmt.Sprintf("not enough space in buffer: %d/%d", requestedSize, 0) {
		t.Errorf("Expected 'not enough space in buffer: %d/%d, got %s", requestedSize, 0, err)
	}

	newBuffer := NewBuffer(10, startPositionOffet)

	bytesToOverflow = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	_, err = newBuffer.Write(bytesToOverflow)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	requestedSize = len(bytesToOverflow)

	if err.Error() != fmt.Sprintf("write data exceeds buffer size: %d", requestedSize) {
		t.Errorf("Expected 'write data exceeds buffer size: %d', got %s", requestedSize, err)
	}
}

func TestRead(t *testing.T) {
	buffer := NewBuffer(10, startPositionOffet)

	bytesToWrite := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	buffer.Write(bytesToWrite)

	testBuffer := make([]byte, 5)

	_, err := buffer.ReadAt(testBuffer, startPositionOffet+0)

	for i, b := range testBuffer {
		if b != bytesToWrite[i] {
			t.Errorf("Expected %d, got %d", bytesToWrite[i], b)
		}
	}

	if buffer.readPosition.Load() != 5 {
		t.Errorf("Expected 5, got %d", buffer.readPosition.Load())
	}

	_, err = buffer.ReadAt(testBuffer, startPositionOffet+5)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

	for i, b := range testBuffer {
		i += 5

		if b != bytesToWrite[i] {
			t.Errorf("Expected %d, got %d", bytesToWrite[i], b)
		}
	}

	if buffer.readPosition.Load() != 0 {
		t.Errorf("Expected 0, got %d", buffer.readPosition.Load())
	}

	buffer.Write([]byte{11, 12, 13, 14})

	testBuffer = make([]byte, 5)
	n, err := buffer.ReadAt(testBuffer, startPositionOffet+12) // = len(testBuffer) + 2 = 7
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

	if buffer.readPosition.Load() != int64(n+2) {
		t.Errorf("Expected %d, got %d", n+2, buffer.readPosition.Load())
	}
}

func TestWriteRead(t *testing.T) {
	buffer := NewBuffer(10, startPositionOffet)
	testBuffer := make([]byte, 5)

	bytes := []byte{0, 1, 2, 3, 4}
	buffer.Write(bytes)
	// t.Logf("Buffer: %v", buffer.data)

	_, err := buffer.ReadAt(testBuffer, startPositionOffet+0)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

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

	_, err = buffer.ReadAt(testBuffer, startPositionOffet+5)
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

	_, err = buffer.ReadAt(testBuffer, startPositionOffet+10)
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
	buffer := NewBuffer(10, startPositionOffet)
	var bytes []byte
	var testBuffer []byte
	var expectedWrittenData []byte
	var expectedReadData []byte

	bytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	buffer.Write(bytes)
	// t.Logf("Buffer: %v", buffer.data)

	testBuffer = make([]byte, 5)
	_, err := buffer.ReadAt(testBuffer, startPositionOffet+0)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

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
	_, err = buffer.ReadAt(testBuffer, startPositionOffet+5)
	// t.Logf("Test buffer: %v", testBuffer)

	expectedReadData = []byte{5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

	for i, b := range testBuffer {
		if b != expectedReadData[i] {
			t.Errorf("Read: Expected %d, got %d", expectedReadData[i], b)
		}
	}

	t.Log("Segment 2 -- Complete")
}

// Helper to check if a position is in the buffer and log errors if expectations are not met.
func checkPosition(t *testing.T, buffer *Buffer, position int64, expected bool) {
	actual := buffer.IsPositionInBufferSync(position)
	if actual != expected {
		t.Errorf("At position %d: expected %v, got %v", position, expected, actual)
	}
}

// Helper to write data into the buffer and verify positions.
func writeAndCheck(t *testing.T, buffer *Buffer, data []byte, expectedPositions map[int64]bool) {
	buffer.Write(data)
	for pos, expected := range expectedPositions {
		checkPosition(t, buffer, pos, expected)
	}
}

// Main test function for buffer position checks.
func TestIsPositionInBuffer(t *testing.T) {
	// Case 1: Test with buffer starting at position 0.
	buffer := NewBuffer(10, 0)
	fmt.Println("Testing with start position 0")

	// Check position 0 before writing.
	checkPosition(t, buffer, 0, false)

	// Write one byte and verify positions.
	writeAndCheck(t, buffer, []byte{1}, map[int64]bool{
		0: true, // Position 0 should now be valid.
		1: false,
	})

	// Write more data and verify positions.
	writeAndCheck(t, buffer, []byte{2, 3, 4, 5, 6, 7, 8, 9, 10}, map[int64]bool{
		0:  true,  // Data at position 0 should still be present.
		1:  true,  // Data extends to position 1.
		10: false, // Position 10 should be outside the buffer.
		11: false, // Position 11 should also be outside.
	})

	// Read some data from the buffer and verify positions are updated.
	_, err := buffer.ReadAt(make([]byte, 9), 0)
	if err != nil {
		t.Errorf("Expected nil error on read, got %s", err)
	}

	// Write more data and verify positions.
	writeAndCheck(t, buffer, []byte{11, 12, 13, 14, 15}, map[int64]bool{
		11: true,
		0:  false, // Data at position 0 has been read, so it should now be invalid.
		1:  false,
		14: true,
		15: false,
	})

	// Read more and verify positions again.
	_, err = buffer.ReadAt(make([]byte, 5), 11)
	if err != nil {
		t.Errorf("Expected nil error on read, got %s", err)
	}

	// Position 11 should be invalid after reading.
	checkPosition(t, buffer, 11, false)

	// Case 2: Test with buffer starting at a non-zero position.
	startOffset := int64(34324)
	buffer = NewBuffer(10, startOffset)
	fmt.Println("Testing with start position", startOffset)

	// Check position 0 before writing.
	checkPosition(t, buffer, startOffset+0, false)

	// Write one byte and verify positions.
	writeAndCheck(t, buffer, []byte{1}, map[int64]bool{
		startOffset + 0: true, // Position 0 should now be valid.
		startOffset + 1: false,
	})

	// Write more data and verify positions.
	writeAndCheck(t, buffer, []byte{2, 3, 4, 5, 6, 7, 8, 9, 10}, map[int64]bool{
		startOffset + 0:  true,  // Data at position 0 should still be present.
		startOffset + 1:  true,  // Data extends to position 1.
		startOffset + 10: false, // Position 10 should be outside the buffer.
		startOffset + 11: false, // Position 11 should also be outside.
	})

	// Read some data from the buffer and verify positions are updated.
	_, err = buffer.ReadAt(make([]byte, 9), startOffset+0)
	if err != nil {
		t.Errorf("Expected nil error on read, got %s", err)
	}

	// Write more data and verify positions.
	writeAndCheck(t, buffer, []byte{11, 12, 13, 14, 15}, map[int64]bool{
		startOffset + 11: true,
		startOffset + 0:  false, // Data at position 0 has been read, so it should now be invalid.
		startOffset + 1:  false,
		startOffset + 14: true,
		startOffset + 15: false,
	})

	// Read more and verify positions again.
	_, err = buffer.ReadAt(make([]byte, 5), startOffset+11)
	if err != nil {
		t.Errorf("Expected nil error on read, got %s", err)
	}

	// Position 11 should be invalid after reading.
	checkPosition(t, buffer, startOffset+11, false)
}

// Not good yet
func TestGetBytesToOverwrite(t *testing.T) {
	buffer := NewBuffer(10, startPositionOffet)

	// Initial state: empty buffer
	bytesToOverwrite := buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 10 {
		t.Errorf("Expected 10, got %d", bytesToOverwrite)
	}

	// Single Write
	fmt.Println()
	buffer.Write([]byte{1, 2, 3, 4, 5})
	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 5 {
		t.Errorf("Expected 5 after writing 5 bytes, got %d", bytesToOverwrite)
	}

	// Single Read
	fmt.Println()
	buffer.ReadAt(make([]byte, 2), startPositionOffet+0) // Reading 2 bytes
	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 7 {
		t.Errorf("Expected 7 after reading 2 bytes, got %d", bytesToOverwrite)
	}

	// Overwriting: Write 3 more bytes
	fmt.Println()
	buffer.Write([]byte{6, 7, 8})
	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 4 {
		t.Errorf("Expected 4 after writing 3 more bytes, got %d", bytesToOverwrite)
	}

	// Write until the buffer is full
	fmt.Println()
	buffer.Write([]byte{9, 10, 1, 2}) // This will fill the buffer completely
	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 0 {
		t.Errorf("Expected 0 after filling the buffer, got %d", bytesToOverwrite)
	}

	// Read 5 bytes from the buffer w12 - r7
	fmt.Println()
	buffer.ReadAt(make([]byte, 5), startPositionOffet+2)
	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 5 {
		t.Errorf("Expected 5 after reading 5 bytes, got %d", bytesToOverwrite)
	}

	fmt.Println()
	_, err := buffer.ReadAt(make([]byte, 3), startPositionOffet+8)
	if err != nil {
		t.Errorf("Expected nil, got %s", err)
	}

	bytesToOverwrite = buffer.GetBytesToOverwriteSync()
	if bytesToOverwrite != 9 {
		t.Errorf("Expected 9 after reading 3 bytes, got %d", bytesToOverwrite)
	}
}
