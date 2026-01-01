package png

import (
	"bytes"
	"testing"
)

func TestWriteIEND(t *testing.T) {
	var buf bytes.Buffer
	err := WriteIEND(&buf)
	if err != nil {
		t.Fatalf("WriteIEND failed: %v", err)
	}

	data := buf.Bytes()

	if len(data) != 12 {
		t.Fatalf("IEND chunk should be 12 bytes, got %d", len(data))
	}

	if data[0] != 0 || data[1] != 0 || data[2] != 0 || data[3] != 0 {
		t.Fatalf("IEND length should be 0, got %x", data[0:4])
	}

	if string(data[4:8]) != "IEND" {
		t.Fatalf("IEND type should be 'IEND', got %s", string(data[4:8]))
	}

	expectedCRC := uint32(0xAE426082)
	crc := uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	if crc != expectedCRC {
		t.Fatalf("IEND CRC should be 0x%08X, got 0x%08X", expectedCRC, crc)
	}
}
