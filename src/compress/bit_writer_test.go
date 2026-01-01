package compress

import (
	"bytes"
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
