package png

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestWriteIDAT_RGB(t *testing.T) {
	// 1x1 RGB image: single red pixel
	pixels := []byte{0xFF, 0x00, 0x00}

	var buf bytes.Buffer
	err := WriteIDAT(&buf, pixels, 1, 1, ColorRGB)
	if err != nil {
		t.Fatalf("WriteIDAT() error = %v", err)
	}

	// Verify chunk structure
	data := buf.Bytes()
	if len(data) < 12 {
		t.Fatalf("IDAT chunk too short: %d bytes", len(data))
	}

	// Check length field (big-endian) - should be reasonable (at least zlib header + footer)
	length := binary.BigEndian.Uint32(data[0:4])
	if length < 6 {
		t.Errorf("chunk length = %d, want at least 6 (zlib header + footer)", length)
	}

	// Check type field
	typeStr := string(data[4:8])
	if typeStr != "IDAT" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IDAT")
	}

	// Verify zlib header (0x78 0x9C for DEFLATE, level 2, 32K window)
	zlibHeader := data[8:10]
	if zlibHeader[0] != 0x78 {
		t.Errorf("zlib CMF = 0x%02X, want 0x78", zlibHeader[0])
	}
	if zlibHeader[1] != 0x9C {
		t.Errorf("zlib FLG = 0x%02X, want 0x9C", zlibHeader[1])
	}

	// Verify CRC
	crc := binary.BigEndian.Uint32(data[len(data)-4:])
	typeAndData := append([]byte("IDAT"), data[8:len(data)-4]...)
	expectedCRC := compress.CRC32(typeAndData)
	if crc != expectedCRC {
		t.Errorf("CRC = 0x%08X, want 0x%08X", crc, expectedCRC)
	}
}

func TestWriteIDAT_RGBA(t *testing.T) {
	// 1x1 RGBA image: red with full alpha
	pixels := []byte{0xFF, 0x00, 0x00, 0xFF}

	var buf bytes.Buffer
	err := WriteIDAT(&buf, pixels, 1, 1, ColorRGBA)
	if err != nil {
		t.Fatalf("WriteIDAT() error = %v", err)
	}

	data := buf.Bytes()

	// Check type field
	typeStr := string(data[4:8])
	if typeStr != "IDAT" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IDAT")
	}

	// Verify zlib header
	zlibHeader := data[8:10]
	if zlibHeader[0] != 0x78 || zlibHeader[1] != 0x9C {
		t.Errorf("unexpected zlib header: %v", zlibHeader)
	}
}

func TestWriteIDAT_2x2RGB(t *testing.T) {
	// 2x2 RGB image
	pixels := []byte{
		0xFF, 0x00, 0x00, // (0,0) red
		0x00, 0xFF, 0x00, // (1,0) green
		0x00, 0x00, 0xFF, // (0,1) blue
		0xFF, 0xFF, 0x00, // (1,1) yellow
	}

	var buf bytes.Buffer
	err := WriteIDAT(&buf, pixels, 2, 2, ColorRGB)
	if err != nil {
		t.Fatalf("WriteIDAT() error = %v", err)
	}

	data := buf.Bytes()
	typeStr := string(data[4:8])
	if typeStr != "IDAT" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IDAT")
	}

	// Verify zlib header
	zlibHeader := data[8:10]
	if zlibHeader[0] != 0x78 || zlibHeader[1] != 0x9C {
		t.Errorf("unexpected zlib header: %v", zlibHeader)
	}
}

func TestWriteIDAT_InvalidDimensions(t *testing.T) {
	pixels := []byte{0xFF, 0x00, 0x00}

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"zero width", 0, 1},
		{"zero height", 1, 0},
		{"negative width", -1, 1},
		{"negative height", 1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteIDAT(&buf, pixels, tt.width, tt.height, ColorRGB)
			if err != ErrInvalidDimensions {
				t.Errorf("WriteIDAT() error = %v, want %v", err, ErrInvalidDimensions)
			}
		})
	}
}

func TestWriteIDAT_WrongPixelCount(t *testing.T) {
	// 1x1 RGB image should have 3 bytes, but we provide 6
	pixels := []byte{0xFF, 0x00, 0x00, 0xFF, 0x00, 0x00}

	var buf bytes.Buffer
	err := WriteIDAT(&buf, pixels, 1, 1, ColorRGB)
	if err == nil {
		t.Errorf("WriteIDAT() expected error for wrong pixel count, got nil")
	}
}

func TestIDATDataBytes(t *testing.T) {
	// 1x1 RGB image
	pixels := []byte{0xFF, 0x00, 0x00}

	data, err := IDATDataBytes(pixels, 1, 1, ColorRGB)
	if err != nil {
		t.Fatalf("IDATDataBytes() error = %v", err)
	}

	// Verify zlib header
	if len(data) < 6 {
		t.Fatalf("IDAT data too short: %d bytes", len(data))
	}

	if data[0] != 0x78 {
		t.Errorf("zlib CMF = 0x%02X, want 0x78", data[0])
	}
	if data[1] != 0x9C {
		t.Errorf("zlib FLG = 0x%02X, want 0x9C", data[1])
	}

	// Decompress and verify data
	zlibReader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}
	defer zlibReader.Close()

	decompressed := make([]byte, 100)
	n, err := zlibReader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	// Build expected scanline data with filter byte 0
	expectedScanlineData := []byte{0x00, 0xFF, 0x00, 0x00}
	if !bytes.Equal(decompressed[:n], expectedScanlineData) {
		t.Errorf("decompressed data = %v, want %v", decompressed[:n], expectedScanlineData)
	}

	// Verify Adler32 footer
	adler := binary.BigEndian.Uint32(data[len(data)-4:])
	expectedAdler := compress.Adler32(expectedScanlineData)
	if adler != expectedAdler {
		t.Errorf("Adler32 = 0x%08X, want 0x%08X", adler, expectedAdler)
	}
}

func TestExpectedIDATSize(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		colorType ColorType
		minSize   int
	}{
		{
			name:      "1x1 RGB",
			width:     1,
			height:    1,
			colorType: ColorRGB,
			minSize:   6, // zlib header (2) + minimum DEFLATE data + adler32 (4)
		},
		{
			name:      "1x1 RGBA",
			width:     1,
			height:    1,
			colorType: ColorRGBA,
			minSize:   6,
		},
		{
			name:      "2x2 RGB",
			width:     2,
			height:    2,
			colorType: ColorRGB,
			minSize:   6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpectedIDATSize(tt.width, tt.height, tt.colorType)
			if got < tt.minSize {
				t.Errorf("ExpectedIDATSize(%d, %d, %d) = %d, want at least %d",
					tt.width, tt.height, tt.colorType, got, tt.minSize)
			}
		})
	}
}

func TestWriteIDAT_CompressionReducesSize(t *testing.T) {
	// Create a repetitive image that should compress well
	width, height := 10, 10
	bpp := 3 // RGB
	repetitivePixel := []byte{0xFF, 0x00, 0x00}
	pixels := make([]byte, width*height*bpp)
	for i := 0; i < width*height; i++ {
		copy(pixels[i*bpp:], repetitivePixel)
	}

	data, err := IDATDataBytes(pixels, width, height, ColorRGB)
	if err != nil {
		t.Fatalf("IDATDataBytes() error = %v", err)
	}

	// Build expected scanline data using filter selection
	expectedScanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		rowStart := y * width * bpp
		row := pixels[rowStart : rowStart+width*bpp]
		filterType, filteredRow := SelectFilter(row, prevRow, bpp)
		expectedScanlineData = append(expectedScanlineData, byte(filterType))
		expectedScanlineData = append(expectedScanlineData, filteredRow...)
		prevRow = row
	}

	uncompressedSize := len(expectedScanlineData)
	compressedSize := len(data) - 6 // subtract zlib header (2) + Adler32 (4)

	// Compressed size should be smaller than uncompressed for repetitive data
	if compressedSize >= uncompressedSize {
		t.Errorf("compression didn't reduce size: compressed=%d, uncompressed=%d",
			compressedSize, uncompressedSize)
	}

	// Verify decompression
	zlibReader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}
	defer zlibReader.Close()

	decompressed := make([]byte, uncompressedSize+100)
	n, err := zlibReader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], expectedScanlineData) {
		t.Errorf("decompressed data doesn't match expected scanline data")
	}
}

func TestWriteIDAT_Grayscale(t *testing.T) {
	// 2x1 grayscale image
	pixels := []byte{0x80, 0x40}

	var buf bytes.Buffer
	err := WriteIDAT(&buf, pixels, 2, 1, ColorGrayscale)
	if err != nil {
		t.Fatalf("WriteIDAT() error = %v", err)
	}

	data := buf.Bytes()
	typeStr := string(data[4:8])
	if typeStr != "IDAT" {
		t.Errorf("chunk type = %q, want %q", typeStr, "IDAT")
	}
}

func TestIDATDataBytes_matchesWriteIDAT(t *testing.T) {
	pixels := []byte{
		0xFF, 0x00, 0x00, 0x00, 0xFF, 0x00, // row 0: 2 RGB pixels
		0x00, 0x00, 0xFF, 0xFF, 0xFF, 0x00, // row 1: 2 RGB pixels
	}

	dataBytes, err := IDATDataBytes(pixels, 2, 2, ColorRGB)
	if err != nil {
		t.Fatalf("IDATDataBytes() error = %v", err)
	}

	var buf bytes.Buffer
	err = WriteIDAT(&buf, pixels, 2, 2, ColorRGB)
	if err != nil {
		t.Fatalf("WriteIDAT() error = %v", err)
	}

	// Extract just the chunk data (skip length + type + CRC)
	writeData := buf.Bytes()[8 : len(buf.Bytes())-4]

	if !bytes.Equal(dataBytes, writeData) {
		t.Errorf("IDATDataBytes() = %v, WriteIDAT() data = %v", dataBytes, writeData)
	}
}
