package png

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestNewIHDRData(t *testing.T) {
	ihdr, err := NewIHDRData(1, 1, 8, 2)
	if err != nil {
		t.Errorf("NewIHDRData() error = %v, want nil", err)
	}

	if ihdr == nil {
		t.Fatal("NewIHDRData() returned nil")
	}

	if ihdr.Width != 1 {
		t.Errorf("ihdr.Width = %d, want 1", ihdr.Width)
	}

	if ihdr.Height != 1 {
		t.Errorf("ihdr.Height = %d, want 1", ihdr.Height)
	}

	if ihdr.BitDepth != 8 {
		t.Errorf("ihdr.BitDepth = %d, want 8", ihdr.BitDepth)
	}

	if ihdr.ColorType != ColorRGB {
		t.Errorf("ihdr.ColorType = %d, want %d", ihdr.ColorType, ColorRGB)
	}
}

func TestIHDRBytes(t *testing.T) {
	ihdr, err := NewIHDRData(1, 1, 8, 2)
	if err != nil {
		t.Fatalf("NewIHDRData() error = %v", err)
	}

	bytes := ihdr.Bytes()

	if len(bytes) != 13 {
		t.Errorf("ihdr.Bytes() length = %d, want 13", len(bytes))
	}

	width := binary.BigEndian.Uint32(bytes[0:4])
	if width != 1 {
		t.Errorf("width field = %d, want 1", width)
	}

	height := binary.BigEndian.Uint32(bytes[4:8])
	if height != 1 {
		t.Errorf("height field = %d, want 1", height)
	}

	if bytes[8] != 8 {
		t.Errorf("bit depth field = %d, want 8", bytes[8])
	}

	if bytes[9] != 2 {
		t.Errorf("color type field = %d, want 2", bytes[9])
	}

	if bytes[10] != 0 {
		t.Errorf("compression field = %d, want 0", bytes[10])
	}

	if bytes[11] != 0 {
		t.Errorf("filter field = %d, want 0", bytes[11])
	}

	if bytes[12] != 0 {
		t.Errorf("interlace field = %d, want 0", bytes[12])
	}
}

func TestIHDRValidate(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		bitDepth  uint8
		colorType uint8
		wantErr   bool
	}{
		{
			name:      "valid RGB 8-bit",
			width:     100,
			height:    100,
			bitDepth:  8,
			colorType: 2,
			wantErr:   false,
		},
		{
			name:      "valid RGBA 8-bit",
			width:     50,
			height:    50,
			bitDepth:  8,
			colorType: 6,
			wantErr:   false,
		},
		{
			name:      "zero width",
			width:     0,
			height:    100,
			bitDepth:  8,
			colorType: 2,
			wantErr:   true,
		},
		{
			name:      "zero height",
			width:     100,
			height:    0,
			bitDepth:  8,
			colorType: 2,
			wantErr:   true,
		},
		{
			name:      "invalid bit depth for RGB",
			width:     100,
			height:    100,
			bitDepth:  4,
			colorType: 2,
			wantErr:   true,
		},
		{
			name:      "invalid color type",
			width:     100,
			height:    100,
			bitDepth:  8,
			colorType: 99,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ihdr, err := NewIHDRData(tt.width, tt.height, tt.bitDepth, tt.colorType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewIHDRData() error = nil, want error")
				}
				if ihdr != nil {
					t.Errorf("NewIHDRData() returned non-nil ihdr when error expected")
				}
			} else {
				if err != nil {
					t.Errorf("NewIHDRData() error = %v, want nil", err)
				}
				if ihdr == nil {
					t.Errorf("NewIHDRData() returned nil ihdr")
				}
			}
		})
	}
}

func TestIHDRBytesLargeDimensions(t *testing.T) {
	ihdr, err := NewIHDRData(1000, 2000, 8, 2)
	if err != nil {
		t.Fatalf("NewIHDRData() error = %v", err)
	}

	bytes := ihdr.Bytes()

	width := binary.BigEndian.Uint32(bytes[0:4])
	if width != 1000 {
		t.Errorf("width field = %d, want 1000", width)
	}

	height := binary.BigEndian.Uint32(bytes[4:8])
	if height != 2000 {
		t.Errorf("height field = %d, want 2000", height)
	}
}

func TestWriteIHDR(t *testing.T) {
	ihdr, err := NewIHDRData(1, 1, 8, 2)
	if err != nil {
		t.Fatalf("NewIHDRData() error = %v", err)
	}

	var buf bytes.Buffer
	err = WriteIHDR(&buf, ihdr)
	if err != nil {
		t.Errorf("WriteIHDR() error = %v, want nil", err)
	}

	writtenBytes := buf.Bytes()

	if len(writtenBytes) != 25 {
		t.Errorf("WriteIHDR() wrote %d bytes, want 25 (4 length + 4 type + 13 data + 4 CRC)", len(writtenBytes))
	}

	length := binary.BigEndian.Uint32(writtenBytes[0:4])
	if length != 13 {
		t.Errorf("chunk length = %d, want 13", length)
	}

	typeStr := string(writtenBytes[4:8])
	if typeStr != "IHDR" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IHDR")
	}

	dataPart := writtenBytes[8:21]
	expectedData := ihdr.Bytes()
	if !bytes.Equal(dataPart, expectedData) {
		t.Errorf("chunk data = %v, want %v", dataPart, expectedData)
	}

	crc := binary.BigEndian.Uint32(writtenBytes[21:25])
	typeBytes := []byte("IHDR")
	combined := append(typeBytes, expectedData...)
	expectedCRC := compress.CRC32(combined)
	if crc != expectedCRC {
		t.Errorf("chunk CRC = 0x%08x, want 0x%08x", crc, expectedCRC)
	}
}

func TestWriteIHDRLargeImage(t *testing.T) {
	ihdr, err := NewIHDRData(1000, 2000, 8, 6)
	if err != nil {
		t.Fatalf("NewIHDRData() error = %v", err)
	}

	var buf bytes.Buffer
	err = WriteIHDR(&buf, ihdr)
	if err != nil {
		t.Errorf("WriteIHDR() error = %v, want nil", err)
	}

	writtenBytes := buf.Bytes()

	length := binary.BigEndian.Uint32(writtenBytes[0:4])
	if length != 13 {
		t.Errorf("chunk length = %d, want 13", length)
	}

	typeStr := string(writtenBytes[4:8])
	if typeStr != "IHDR" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IHDR")
	}
}
