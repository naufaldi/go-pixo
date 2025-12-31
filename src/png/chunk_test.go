package png

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestChunkCreation(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00}
	chunk := &Chunk{
		chunkType: ChunkIHDR,
		Data:      data,
	}

	if chunk.Type() != string(ChunkIHDR) {
		t.Errorf("chunk.Type() = %v, want %v", chunk.Type(), string(ChunkIHDR))
	}

	if chunk.Len() != len(data) {
		t.Errorf("chunk.Len() = %d, want %d", chunk.Len(), len(data))
	}

	crc := chunk.CRC()
	if crc == 0 {
		t.Errorf("chunk.CRC() = 0x%08x, should not be 0 (CRC32 computed)", crc)
	}
}

func TestChunkLen(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int
	}{
		{
			name: "empty data",
			data: []byte{},
			want: 0,
		},
		{
			name: "IHDR data",
			data: []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00},
			want: 13,
		},
		{
			name: "IEND data",
			data: []byte{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				chunkType: ChunkIHDR,
				Data:      tt.data,
			}
			if got := chunk.Len(); got != tt.want {
				t.Errorf("chunk.Len() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestChunkType(t *testing.T) {
	tests := []struct {
		name      string
		chunkType ChunkType
		want      ChunkType
	}{
		{
			name:      "IHDR",
			chunkType: ChunkIHDR,
			want:      ChunkIHDR,
		},
		{
			name:      "IDAT",
			chunkType: ChunkIDAT,
			want:      ChunkIDAT,
		},
		{
			name:      "IEND",
			chunkType: ChunkIEND,
			want:      ChunkIEND,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk := &Chunk{
				chunkType: tt.chunkType,
				Data:      []byte{},
			}
			if got := chunk.Type(); got != string(tt.want) {
				t.Errorf("chunk.Type() = %v, want %v", got, string(tt.want))
			}
		})
	}
}

func TestChunkCRC(t *testing.T) {
	chunk := &Chunk{
		chunkType: ChunkIHDR,
		Data:      []byte{0x01, 0x02, 0x03},
	}

	typeBytes := []byte("IHDR")
	combined := append(typeBytes, chunk.Data...)
	expected := compress.CRC32(combined)

	if got := chunk.CRC(); got != expected {
		t.Errorf("chunk.CRC() = 0x%08x, want 0x%08x", got, expected)
	}
}

func TestChunkBytes(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x02, 0x00, 0x00, 0x00}
	chunk := &Chunk{
		chunkType: ChunkIHDR,
		Data:      data,
	}

	chunkBytes := chunk.Bytes()

	if len(chunkBytes) != 4+4+len(data)+4 {
		t.Errorf("chunk.Bytes() length = %d, want %d", len(chunkBytes), 4+4+len(data)+4)
	}

	length := binary.BigEndian.Uint32(chunkBytes[0:4])
	if length != uint32(len(data)) {
		t.Errorf("length field = %d, want %d", length, len(data))
	}

	typeStr := string(chunkBytes[4:8])
	if typeStr != "IHDR" {
		t.Errorf("type field = %q, want %q", typeStr, "IHDR")
	}

	dataPart := chunkBytes[8 : 8+len(data)]
	if !bytes.Equal(dataPart, data) {
		t.Errorf("data field = %v, want %v", dataPart, data)
	}

	crc := binary.BigEndian.Uint32(chunkBytes[8+len(data):])
	expectedCRC := chunk.CRC()
	if crc != expectedCRC {
		t.Errorf("CRC field = 0x%08x, want 0x%08x", crc, expectedCRC)
	}
}

func TestChunkWriteTo(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x01}
	chunk := &Chunk{
		chunkType: ChunkIEND,
		Data:      data,
	}

	var buf bytes.Buffer
	n, err := chunk.WriteTo(&buf)

	if err != nil {
		t.Errorf("chunk.WriteTo() error = %v, want nil", err)
	}

	expectedBytes := chunk.Bytes()
	if int64(len(expectedBytes)) != n {
		t.Errorf("chunk.WriteTo() wrote %d bytes, want %d", n, len(expectedBytes))
	}

	if !bytes.Equal(buf.Bytes(), expectedBytes) {
		t.Errorf("chunk.WriteTo() output = %v, want %v", buf.Bytes(), expectedBytes)
	}
}

func TestChunkBytesIEND(t *testing.T) {
	chunk := &Chunk{
		chunkType: ChunkIEND,
		Data:      []byte{},
	}

	chunkBytes := chunk.Bytes()

	if len(chunkBytes) != 12 {
		t.Errorf("IEND chunk.Bytes() length = %d, want 12", len(chunkBytes))
	}

	length := binary.BigEndian.Uint32(chunkBytes[0:4])
	if length != 0 {
		t.Errorf("IEND length field = %d, want 0", length)
	}

	typeStr := string(chunkBytes[4:8])
	if typeStr != "IEND" {
		t.Errorf("IEND type field = %q, want %q", typeStr, "IEND")
	}

	crc := binary.BigEndian.Uint32(chunkBytes[8:12])
	expectedCRC := compress.CRC32([]byte("IEND"))
	if crc != expectedCRC {
		t.Errorf("IEND CRC field = 0x%08x, want 0x%08x", crc, expectedCRC)
	}
}
