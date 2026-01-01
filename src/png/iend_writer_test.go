package png

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestWriteIEND(t *testing.T) {
	writeIENDBytes := func(t *testing.T) []byte {
		t.Helper()

		var buf bytes.Buffer
		if err := WriteIEND(&buf); err != nil {
			t.Fatalf("WriteIEND() error = %v, want nil", err)
		}

		return buf.Bytes()
	}

	tests := []struct {
		name  string
		check func(t *testing.T, got []byte)
	}{
		{
			name: "writes 12 bytes (4 length + 4 type + 0 data + 4 CRC)",
			check: func(t *testing.T, got []byte) {
				if len(got) != 12 {
					t.Fatalf("WriteIEND() wrote %d bytes, want 12", len(got))
				}
			},
		},
		{
			name: "writes zero length and IEND type",
			check: func(t *testing.T, got []byte) {
				if len(got) < 8 {
					t.Fatalf("WriteIEND() wrote %d bytes, want at least 8", len(got))
				}

				length := binary.BigEndian.Uint32(got[0:4])
				if length != 0 {
					t.Fatalf("length field = %d, want 0", length)
				}

				typeStr := string(got[4:8])
				if typeStr != "IEND" {
					t.Fatalf("type field = %q, want %q", typeStr, "IEND")
				}
			},
		},
		{
			name: "writes correct CRC32 over type bytes",
			check: func(t *testing.T, got []byte) {
				if len(got) != 12 {
					t.Fatalf("WriteIEND() wrote %d bytes, want 12", len(got))
				}

				crc := binary.BigEndian.Uint32(got[8:12])
				expectedCRC := compress.CRC32([]byte("IEND"))
				if crc != expectedCRC {
					t.Fatalf("CRC field = 0x%08x, want 0x%08x", crc, expectedCRC)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writeIENDBytes(t)
			tt.check(t, got)
		})
	}
}
