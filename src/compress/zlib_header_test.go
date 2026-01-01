package compress

import (
	"bytes"
	"testing"
)

func TestWriteCMF(t *testing.T) {
	testCases := []struct {
		windowSize int
		expected   byte
	}{
		{1, 0x08},
		{8, 0x38},
		{32, 0x58},
		{32768, 0xF8},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteCMF(&buf, tc.windowSize)
			if err != nil {
				t.Fatalf("WriteCMF(%d) failed: %v", tc.windowSize, err)
			}
			if buf.Bytes()[0] != tc.expected {
				t.Errorf("WriteCMF(%d) = 0x%02X, want 0x%02X", tc.windowSize, buf.Bytes()[0], tc.expected)
			}
		})
	}
}

func TestWriteCMFInvalidWindowSize(t *testing.T) {
	invalidSizes := []int{0, 3, 5, 100, 65536}

	for _, size := range invalidSizes {
		t.Run("", func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteCMF(&buf, size)
			if err != ErrInvalidWindowSize {
				t.Errorf("WriteCMF(%d) should return ErrInvalidWindowSize, got %v", size, err)
			}
		})
	}
}

func TestWriteFLG(t *testing.T) {
	testCases := []struct {
		name     string
		checksum uint8
	}{
		{"valid checksum 0", 0},
		{"valid checksum 31", 31},
		{"valid checksum 15", 15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteFLG(&buf, tc.checksum)
			if err != nil {
				t.Errorf("WriteFLG() error = %v", err)
			}
			if buf.Len() != 1 {
				t.Errorf("WriteFLG() wrote %d bytes, want 1", buf.Len())
			}
		})
	}
}

func TestZlibHeaderFormat(t *testing.T) {
	var buf bytes.Buffer

	// CMF: window=32, DEFLATE
	err := WriteCMF(&buf, 32)
	if err != nil {
		t.Fatalf("WriteCMF failed: %v", err)
	}

	// FLG: level=2, no dict
	err = WriteFLG(&buf, 0)
	if err != nil {
		t.Fatalf("WriteFLG failed: %v", err)
	}

	if buf.Len() != 2 {
		t.Errorf("Zlib header should be 2 bytes, got %d", buf.Len())
	}

	cmf := buf.Bytes()[0]
	if cmf != 0x58 {
		t.Errorf("CMF = 0x%02X, want 0x58", cmf)
	}
}
