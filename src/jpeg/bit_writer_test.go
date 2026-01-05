package jpeg

import (
	"bytes"
	"testing"
)

func TestBitWriter_Basic(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)

	// Write 0b1010 (10) in 4 bits, then 0b1100 (12) in 4 bits
	// Total: 0b10101100 = 0xAC
	if err := bw.Write(10, 4); err != nil {
		t.Fatal(err)
	}
	if err := bw.Write(12, 4); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 1 {
		t.Errorf("expected 1 byte, got %d", buf.Len())
	}
	if buf.Bytes()[0] != 0xAC {
		t.Errorf("got 0x%X, want 0xAC", buf.Bytes()[0])
	}
}

func TestBitWriter_ByteStuffing(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)

	// Write 0xFF
	if err := bw.Write(0xFF, 8); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 2 {
		t.Errorf("expected 2 bytes (stuffed), got %d", buf.Len())
	}
	if buf.Bytes()[0] != 0xFF || buf.Bytes()[1] != 0x00 {
		t.Errorf("got 0x%X 0x%X, want 0xFF 0x00", buf.Bytes()[0], buf.Bytes()[1])
	}
}

func TestBitWriter_Flush(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)

	// Write 0b10 (2) in 2 bits
	if err := bw.Write(2, 2); err != nil {
		t.Fatal(err)
	}

	// Flush: should pad with 1s (6 bits of 1s)
	// 0b10 + 0b111111 = 0b10111111 = 0xBF
	if err := bw.Flush(); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 1 {
		t.Errorf("expected 1 byte, got %d", buf.Len())
	}
	if buf.Bytes()[0] != 0xBF {
		t.Errorf("got 0x%X, want 0xBF", buf.Bytes()[0])
	}
}

func TestBitWriter_MultiByte(t *testing.T) {
	buf := new(bytes.Buffer)
	bw := NewBitWriter(buf)

	// Write 0xDEADBEEF in 32 bits
	if err := bw.Write(0xDEADBEEF, 32); err != nil {
		t.Fatal(err)
	}

	expected := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("got %X, want %X", buf.Bytes(), expected)
	}
}
