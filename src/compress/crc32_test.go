package compress

import (
	"hash/crc32"
	"testing"
)

func TestCRC32(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint32
	}{
		{
			name:     "empty",
			data:     []byte{},
			expected: 0x00000000,
		},
		{
			name:     "test string",
			data:     []byte("test"),
			expected: 0xd87f7e0c,
		},
		{
			name:     "IHDR type",
			data:     []byte("IHDR"),
			expected: 0xa8a1ae0a,
		},
		{
			name:     "IEND type",
			data:     []byte("IEND"),
			expected: 0xae426082,
		},
		{
			name:     "PNG signature",
			data:     []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a},
			expected: 0x7a0709a4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CRC32(tt.data)
			if result != tt.expected {
				t.Errorf("CRC32(%v) = 0x%08x, want 0x%08x", tt.data, result, tt.expected)
			}
		})
	}
}

func TestCRC32Streaming(t *testing.T) {
	data := []byte("test")

	hasher := NewCRC32()
	hasher.Write(data)
	result := hasher.Sum32()

	expected := CRC32(data)
	if result != expected {
		t.Errorf("Streaming CRC32 = 0x%08x, want 0x%08x", result, expected)
	}
}

func TestCRC32ChunkTypeAndData(t *testing.T) {
	chunkType := []byte("IHDR")
	chunkData := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00}

	combined := append(chunkType, chunkData...)
	result := CRC32(combined)

	table := crc32.MakeTable(crc32.IEEE)
	hasher := crc32.New(table)
	hasher.Write(chunkType)
	hasher.Write(chunkData)
	expected := hasher.Sum32()

	if result != expected {
		t.Errorf("CRC32(chunkType + chunkData) = 0x%08x, want 0x%08x", result, expected)
	}
}
