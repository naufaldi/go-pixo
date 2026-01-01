package compress

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWriteStoredBlockHeader(t *testing.T) {
	tests := []struct {
		name   string
		final  bool
		expect byte
	}{
		{
			name:   "final block",
			final:  true,
			expect: 0x01, // BFINAL=1, type=000
		},
		{
			name:   "non-final block",
			final:  false,
			expect: 0x00, // BFINAL=0, type=000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteStoredBlockHeader(&buf, tt.final)
			if err != nil {
				t.Fatalf("WriteStoredBlockHeader(%v) error = %v", tt.final, err)
			}

			if buf.Len() != 1 {
				t.Fatalf("WriteStoredBlockHeader wrote %d bytes, want 1", buf.Len())
			}

			if got := buf.Bytes()[0]; got != tt.expect {
				t.Fatalf("WriteStoredBlockHeader(%v) = 0x%02X, want 0x%02X", tt.final, got, tt.expect)
			}
		})
	}
}

func TestWriteBlockData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "single byte",
			data: []byte{0x41},
		},
		{
			name: "multiple bytes",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
		{
			name: "ASCII string",
			data: []byte("hello world"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteBlockData(&buf, tt.data)
			if err != nil {
				t.Fatalf("WriteBlockData(%v) error = %v", tt.data, err)
			}

			if !bytes.Equal(buf.Bytes(), tt.data) {
				t.Errorf("WriteBlockData(%v) = %v, want %v", tt.data, buf.Bytes(), tt.data)
			}
		})
	}
}

func TestWriteBlockFooter(t *testing.T) {
	tests := []struct {
		name        string
		n           uint32
		expectLEN   uint16
		expectNLEN  uint16
		expectError bool
	}{
		{
			name:       "zero",
			n:          0,
			expectLEN:  0x0000,
			expectNLEN: 0xFFFF,
		},
		{
			name:       "one",
			n:          1,
			expectLEN:  0x0001,
			expectNLEN: 0xFFFE,
		},
		{
			name:       "255",
			n:          255,
			expectLEN:  0x00FF,
			expectNLEN: 0xFF00,
		},
		{
			name:       "256",
			n:          256,
			expectLEN:  0x0100,
			expectNLEN: 0xFEFF,
		},
		{
			name:       "65535",
			n:          65535,
			expectLEN:  0xFFFF,
			expectNLEN: 0x0000,
		},
		{
			name:        "exceeds maximum",
			n:           65536,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteBlockFooter(&buf, tt.n)
			if tt.expectError {
				if err != ErrInvalidBlockSize {
					t.Fatalf("WriteBlockFooter(%d) error = %v, want %v", tt.n, err, ErrInvalidBlockSize)
				}
				return
			}
			if err != nil {
				t.Fatalf("WriteBlockFooter(%d) error = %v, want nil", tt.n, err)
			}

			if buf.Len() != 4 {
				t.Fatalf("WriteBlockFooter wrote %d bytes, want 4", buf.Len())
			}

			len := binary.LittleEndian.Uint16(buf.Bytes()[0:2])
			nlen := binary.LittleEndian.Uint16(buf.Bytes()[2:4])

			if len != tt.expectLEN {
				t.Errorf("LEN = 0x%04X, want 0x%04X", len, tt.expectLEN)
			}

			if nlen != tt.expectNLEN {
				t.Errorf("NLEN = 0x%04X, want 0x%04X", nlen, tt.expectNLEN)
			}

			// Verify NLEN is one's complement of LEN
			if nlen != ^len {
				t.Errorf("NLEN (0x%04X) should be one's complement of LEN (0x%04X)", nlen, len)
			}
		})
	}
}

func TestWriteStoredBlock(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		final      bool
		expectLen  int
		expectByte byte
	}{
		{
			name:      "empty non-final",
			data:      []byte{},
			final:     false,
			expectLen: 5, // 1 header + 4 footer (LEN=0, NLEN=0xFFFF) + 0 data
		},
		{
			name:      "empty final",
			data:      []byte{},
			final:     true,
			expectLen: 5,
		},
		{
			name:      "single byte final",
			data:      []byte{0xAB},
			final:     true,
			expectLen: 6, // 1 + 4 + 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteStoredBlock(&buf, tt.data, tt.final)
			if err != nil {
				t.Fatalf("WriteStoredBlock(%v, %v) error = %v", tt.data, tt.final, err)
			}

			if buf.Len() != tt.expectLen {
				t.Errorf("WriteStoredBlock wrote %d bytes, want %d", buf.Len(), tt.expectLen)
			}

			// Check header byte
			header := buf.Bytes()[0]
			if tt.final {
				if header != 0x01 {
					t.Errorf("Header = 0x%02X, want 0x01 for final block", header)
				}
			} else {
				if header != 0x00 {
					t.Errorf("Header = 0x%02X, want 0x00 for non-final block", header)
				}
			}
		})
	}
}

func TestStoredBlockBytes(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		final       bool
		expectError bool
	}{
		{
			name:        "valid data",
			data:        []byte{0x01, 0x02, 0x03},
			final:       true,
			expectError: false,
		},
		{
			name:        "empty data",
			data:        []byte{},
			final:       false,
			expectError: false,
		},
		{
			name:        "max size data",
			data:        make([]byte, 65535),
			final:       true,
			expectError: false,
		},
		{
			name:        "exceeds max size",
			data:        make([]byte, 65536),
			final:       true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StoredBlockBytes(tt.data, tt.final)
			if tt.expectError {
				if err != ErrInvalidBlockSize {
					t.Fatalf("StoredBlockBytes(%v, %v) error = %v, want %v", tt.data, tt.final, err, ErrInvalidBlockSize)
				}
				return
			}
			if err != nil {
				t.Fatalf("StoredBlockBytes(%v, %v) error = %v, want nil", tt.data, tt.final, err)
			}

			// Verify header
			if tt.final {
				if result[0] != 0x01 {
					t.Errorf("Header byte = 0x%02X, want 0x01", result[0])
				}
			} else {
				if result[0] != 0x00 {
					t.Errorf("Header byte = 0x%02X, want 0x00", result[0])
				}
			}

			// Verify LEN/NLEN
			expectedLen := uint16(len(tt.data))
			gotLen := binary.LittleEndian.Uint16(result[1:3])
			if gotLen != expectedLen {
				t.Errorf("LEN = %d, want %d", gotLen, expectedLen)
			}

			gotNlen := binary.LittleEndian.Uint16(result[3:5])
			expectedNlen := ^expectedLen
			if gotNlen != expectedNlen {
				t.Errorf("NLEN = 0x%04X, want 0x%04X", gotNlen, expectedNlen)
			}

			// Verify data
			dataResult := result[5:]
			if !bytes.Equal(dataResult, tt.data) {
				t.Errorf("Data = %v, want %v", dataResult, tt.data)
			}

			// Verify total length
			expectedTotal := 1 + 4 + len(tt.data)
			if len(result) != expectedTotal {
				t.Errorf("Total length = %d, want %d", len(result), expectedTotal)
			}
		})
	}
}

func TestStoredBlockBytes_onesComplement(t *testing.T) {
	// Test specific values to verify one's complement relationship
	tests := []struct {
		data []byte
	}{
		{[]byte{0x00}},
		{[]byte{0xFF, 0xFF}},
		{[]byte{0x12, 0x34, 0x56, 0x78}},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result, err := StoredBlockBytes(tt.data, true)
			if err != nil {
				t.Fatalf("StoredBlockBytes error = %v", err)
			}

			len := binary.LittleEndian.Uint16(result[1:3])
			nlen := binary.LittleEndian.Uint16(result[3:5])

			if nlen != ^len {
				t.Errorf("NLEN (0x%04X) is not one's complement of LEN (0x%04X)", nlen, len)
			}
		})
	}
}
