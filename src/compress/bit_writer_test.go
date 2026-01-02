package compress

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestBitWriter_WriteSingleBit(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(1, 1); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0x01}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %v, want %v", buf.Bytes(), expected)
	}
}

func TestBitWriter_WriteMultipleBits(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0b101, 3); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0b00000101}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %08b, want %08b", buf.Bytes()[0], expected[0])
	}
}

func TestBitWriter_WriteLSBFirst(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0b1101, 4); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0b00001101}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %08b, want %08b", buf.Bytes()[0], expected[0])
	}
}

func TestBitWriter_WriteAcrossByteBoundary(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0xFF, 8); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Write(0x01, 1); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0xFF, 0x01}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %v, want %v", buf.Bytes(), expected)
	}
}

func TestBitWriter_WritePartialByte(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0b101, 3); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0b00000101}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %08b, want %08b", buf.Bytes()[0], expected[0])
	}
}

func TestBitWriter_WriteZeroBits(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0xFF, 0); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got %d bytes", buf.Len())
	}
}

func TestBitWriter_FlushEmpty(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got %d bytes", buf.Len())
	}
}

func TestBitWriter_WriteComplexPattern(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	tests := []struct {
		bits uint16
		n    int
	}{
		{0b1, 1},
		{0b0, 1},
		{0b1, 1},
		{0b0, 1},
		{0b1, 1},
		{0b0, 1},
		{0b1, 1},
		{0b0, 1},
		{0b11111111, 8},
	}

	for _, tt := range tests {
		if err := bw.Write(tt.bits, tt.n); err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	expected := []byte{0b01010101, 0xFF}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %08b %08b, want %08b %08b", buf.Bytes()[0], buf.Bytes()[1], expected[0], expected[1])
	}
}

func TestBitWriter_WriteMultipleFullBytes(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	for i := 0; i < 5; i++ {
		if err := bw.Write(0xFF, 8); err != nil {
			t.Fatalf("Write failed at iteration %d: %v", i, err)
		}
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 5 {
		t.Errorf("Expected 5 bytes, got %d", buf.Len())
	}

	for i := 0; i < 5; i++ {
		if buf.Bytes()[i] != 0xFF {
			t.Errorf("Byte %d: expected 0xFF, got 0x%02X", i, buf.Bytes()[i])
		}
	}
}

func TestBitWriter_FlushPartialBits(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0b101, 3); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("Expected no bytes before flush, got %d", buf.Len())
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 1 {
		t.Errorf("Expected 1 byte after flush, got %d", buf.Len())
	}

	expected := byte(0b00000101)
	if buf.Bytes()[0] != expected {
		t.Errorf("got %08b, want %08b", buf.Bytes()[0], expected)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	if buf.Len() != 1 {
		t.Errorf("Expected still 1 byte after second flush, got %d", buf.Len())
	}
}

func TestBitWriter_ErrorPropagation(t *testing.T) {
	expectedError := errors.New("write error")
	errWriter := &errorWriter{err: expectedError}

	bw := NewBitWriter(errWriter)

	if err := bw.Write(0xFF, 8); err != expectedError {
		t.Errorf("Write error = %v, want %v", err, expectedError)
	}

	if err := bw.Flush(); err != expectedError {
		t.Errorf("Flush error = %v, want %v", err, expectedError)
	}
}

type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, e.err
}

func TestBitWriter_WriteThenFlushMultipleTimes(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0b101, 3); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	if err := bw.Write(0b110, 3); err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	expected := []byte{0b00000101, 0b00000110}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %v, want %v", buf.Bytes(), expected)
	}
}

func TestBitWriter_WriteLargeValue(t *testing.T) {
	var buf bytes.Buffer
	bw := NewBitWriter(&buf)

	if err := bw.Write(0xFFFF, 16); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 2 {
		t.Errorf("Expected 2 bytes, got %d", buf.Len())
	}

	expected := []byte{0xFF, 0xFF}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %v, want %v", buf.Bytes(), expected)
	}
}

func TestBitWriter_WriteWithNilWriter(t *testing.T) {
	bw := NewBitWriter(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when writing to nil writer, got no panic")
		}
	}()

	bw.Write(0xFF, 8)
	t.Error("Expected panic, but Write completed")
}

type limitedWriter struct {
	w     io.Writer
	limit int
}

func (l *limitedWriter) Write(p []byte) (n int, err error) {
	if len(p) > l.limit {
		return l.w.Write(p[:l.limit])
	}
	return l.w.Write(p)
}

func TestBitWriter_PartialWrite(t *testing.T) {
	var buf bytes.Buffer
	limited := &limitedWriter{w: &buf, limit: 1}
	bw := NewBitWriter(limited)

	if err := bw.Write(0xFF, 8); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if err := bw.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if buf.Len() != 1 {
		t.Errorf("Expected 1 byte, got %d", buf.Len())
	}
}
