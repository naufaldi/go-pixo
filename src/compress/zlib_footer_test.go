package compress

import (
	"bytes"
	"testing"
)

func TestWriteAdler32Footer(t *testing.T) {
	testCases := []uint32{
		0x00000000,
		0xFFFFFFFF,
		0x12345678,
		0x821AA026,
	}

	for _, checksum := range testCases {
		t.Run("", func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteAdler32Footer(&buf, checksum)
			if err != nil {
				t.Fatalf("WriteAdler32Footer(0x%08X) failed: %v", checksum, err)
			}

			if buf.Len() != 4 {
				t.Errorf("Footer should be 4 bytes, got %d", buf.Len())
			}

			got := uint32(buf.Bytes()[0])<<24 |
				uint32(buf.Bytes()[1])<<16 |
				uint32(buf.Bytes()[2])<<8 |
				uint32(buf.Bytes()[3])
			if got != checksum {
				t.Errorf("Adler32Footer wrote 0x%08X, want 0x%08X", got, checksum)
			}
		})
	}
}
