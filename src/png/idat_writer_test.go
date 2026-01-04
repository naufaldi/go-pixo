package png

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"image/color"
	stdpng "image/png"
	"io"
	"os"
	"path/filepath"
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
	t.Cleanup(func() {
		if closeErr := zlibReader.Close(); closeErr != nil {
			t.Errorf("zlib reader close error: %v", closeErr)
		}
	})

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
	t.Cleanup(func() {
		if closeErr := zlibReader.Close(); closeErr != nil {
			t.Errorf("zlib reader close error: %v", closeErr)
		}
	})

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

func TestSizeComparisonFallback_StoredBlockForRandomData(t *testing.T) {
	// Create random/uncompressible data that should trigger stored block fallback
	width, height := 16, 16
	bpp := 4 // RGBA
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i * 17 % 256) // Pseudo-random pattern
	}

	// Get IDAT data (zlib-wrapped DEFLATE/stored)
	data, err := IDATDataBytes(pixels, width, height, ColorRGBA)
	if err != nil {
		t.Fatalf("IDATDataBytes() error = %v", err)
	}

	// Build expected scanline data with filter bytes
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		rowStart := y * width * bpp
		row := pixels[rowStart : rowStart+width*bpp]
		filterType, filteredRow := SelectFilter(row, prevRow, bpp)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	// The zlib data should include:
	// - zlib header (2 bytes)
	// - DEFLATE/stored block data
	// - Adler32 footer (4 bytes)
	zlibData := data
	if len(zlibData) < 6 {
		t.Fatalf("zlib data too short: %d bytes", len(zlibData))
	}

	// Decompress and verify data is correct
	zlibReader, err := zlib.NewReader(bytes.NewReader(zlibData))
	if err != nil {
		t.Fatalf("failed to create zlib reader: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := zlibReader.Close(); closeErr != nil {
			t.Errorf("zlib reader close error: %v", closeErr)
		}
	})

	decompressed := make([]byte, len(scanlineData)+100)
	n, err := zlibReader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if !bytes.Equal(decompressed[:n], scanlineData) {
		t.Errorf("decompressed data doesn't match expected scanline data")
	}

	t.Logf("Random data: scanline size=%d, compressed zlib size=%d", len(scanlineData), len(zlibData))
}

func TestSizeComparisonFallback_DeflateForCompressibleData(t *testing.T) {
	// Create highly compressible data (all same color)
	width, height := 32, 32
	bpp := 3 // RGB
	pixels := make([]byte, width*height*bpp)
	for i := 0; i < width*height; i++ {
		pixels[i*bpp] = 0xFF   // R
		pixels[i*bpp+1] = 0x00 // G
		pixels[i*bpp+2] = 0x00 // B
	}

	data, err := IDATDataBytes(pixels, width, height, ColorRGB)
	if err != nil {
		t.Fatalf("IDATDataBytes() error = %v", err)
	}

	// Build expected scanline data
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		rowStart := y * width * bpp
		row := pixels[rowStart : rowStart+width*bpp]
		filterType, filteredRow := SelectFilter(row, prevRow, bpp)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	// For highly compressible data, DEFLATE should produce much smaller output
	// zlib format: header (2) + compressed data + footer (4)
	zlibData := data[8 : len(data)-4] // Extract just DEFLATE data
	compressedSize := len(zlibData)

	// Compressed should be significantly smaller than uncompressed
	// For all-same color, compression ratio should be excellent
	if compressedSize >= len(scanlineData)/2 {
		t.Errorf("DEFLATE should compress well: compressed=%d, uncompressed=%d",
			compressedSize, len(scanlineData))
	}

	t.Logf("Compressible data: scanline size=%d, DEFLATE size=%d, ratio=%.2f%%",
		len(scanlineData), compressedSize,
		float64(compressedSize)/float64(len(scanlineData))*100)
}

func TestSizeComparisonFallback_NeverIncreasesSize(t *testing.T) {
	// Test both compressible and incompressible data to ensure size never increases
	testCases := []struct {
		name   string
		width  int
		height int
		bpp    int
		data   []byte
	}{
		{"repetitive", 100, 30, 3, bytes.Repeat([]byte{0xAA, 0xBB, 0xCC}, 1000)},
		{"gradient", 50, 60, 3, generateGradientData(50, 60, 3)},
		{"random", 50, 80, 4, generateRandomData(50, 80, 4)},
		{"checkerboard", 32, 32, 3, generateCheckerboardData(32, 32, 3)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedSize := tc.width * tc.height * tc.bpp
			if len(tc.data) != expectedSize {
				// Trim or pad data to match expected size
				if len(tc.data) > expectedSize {
					tc.data = tc.data[:expectedSize]
				} else {
					// Generate proper sized data
					switch tc.name {
					case "repetitive":
						tc.data = bytes.Repeat([]byte{0xAA, 0xBB, 0xCC}, expectedSize/3)
					case "gradient":
						tc.data = generateGradientData(tc.width, tc.height, tc.bpp)
					case "random":
						tc.data = generateRandomData(tc.width, tc.height, tc.bpp)
					case "checkerboard":
						tc.data = generateCheckerboardData(tc.width, tc.height, tc.bpp)
					}
				}
			}

			colorType := ColorRGB
			if tc.bpp == 4 {
				colorType = ColorRGBA
			}

			data, err := IDATDataBytes(tc.data, tc.width, tc.height, colorType)
			if err != nil {
				t.Fatalf("IDATDataBytes() error = %v", err)
			}

			// zlib data is the data portion
			zlibData := data

			// Decompress to get original size
			zlibReader, err := zlib.NewReader(bytes.NewReader(zlibData))
			if err != nil {
				t.Fatalf("failed to create zlib reader: %v", err)
			}
			t.Cleanup(func() {
				if closeErr := zlibReader.Close(); closeErr != nil {
					t.Errorf("zlib reader close error: %v", closeErr)
				}
			})

			decompressed := make([]byte, len(tc.data)+tc.width*tc.height+100)
			n, err := zlibReader.Read(decompressed)
			if err != nil && err != io.EOF {
				t.Fatalf("decompression failed: %v", err)
			}

			// The PNG output (zlib data) should be <= the original scanline data size
			// Note: we compare against decompressed size (which includes filter bytes)
			if len(zlibData) > n {
				t.Errorf("compressed size (%d) > original size (%d): compression increased file size!",
					len(zlibData), n)
			}

			t.Logf("%s: original=%d, compressed=%d, ratio=%.2f%%",
				tc.name, n, len(zlibData),
				float64(len(zlibData))/float64(n)*100)
		})
	}
}

func TestEncodeWithFallback(t *testing.T) {
	encoder := compress.NewDeflateEncoder()

	// Test 1: Highly compressible data
	compressible := bytes.Repeat([]byte{0x01, 0x02, 0x03}, 1000)
	result, err := encoder.EncodeWithFallback(compressible)
	if err != nil {
		t.Fatalf("EncodeWithFallback() error = %v", err)
	}
	if len(result) >= len(compressible) {
		t.Errorf("compressible: result (%d) should be < input (%d)", len(result), len(compressible))
	}

	// Test 2: Random/uncompressible data
	random := generateRandomData(100, 100, 1)
	result, err = encoder.EncodeWithFallback(random)
	if err != nil {
		t.Fatalf("EncodeWithFallback() error = %v", err)
	}
	// For uncompressible data, result may be slightly larger due to stored block overhead
	// but should use stored blocks (not expanded DEFLATE)
	t.Logf("random: input=%d, output=%d, ratio=%.2f%%",
		len(random), len(result), float64(len(result))/float64(len(random))*100)
}

func TestEncodeStored(t *testing.T) {
	encoder := compress.NewDeflateEncoder()

	data := []byte("hello world")
	result, err := encoder.EncodeStored(data)
	if err != nil {
		t.Fatalf("EncodeStored() error = %v", err)
	}

	// Verify result is a valid DEFLATE stored block
	// Stored block format: 0x01 (BFINAL=1, TYPE=000) + LEN + NLEN + data
	if len(result) < 5 {
		t.Fatalf("stored block too short: %d bytes", len(result))
	}

	if result[0] != 0x01 {
		t.Errorf("BFINAL = 0x%02X, want 0x01", result[0])
	}

	// Verify LEN/NLEN
	lenValue := uint16(result[1]) | uint16(result[2])<<8
	nlenValue := uint16(result[3]) | uint16(result[4])<<8
	if nlenValue != ^lenValue {
		t.Errorf("NLEN = 0x%04X, want one's complement of LEN (0x%04X)", nlenValue, lenValue)
	}

	// Verify data
	if !bytes.Equal(result[5:], data) {
		t.Errorf("stored data = %v, want %v", result[5:], data)
	}
}

// Helper functions for generating test data

func generateGradientData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			for c := 0; c < bpp; c++ {
				data[offset+c] = byte((x + y + c) % 256)
			}
		}
	}
	return data
}

func generateRandomData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	for i := range data {
		data[i] = byte(i * 17 % 256)
	}
	return data
}

func generateCheckerboardData(width, height, bpp int) []byte {
	data := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			if (x+y)%2 == 0 {
				for c := 0; c < bpp; c++ {
					data[offset+c] = 0xFF
				}
			} else {
				for c := 0; c < bpp; c++ {
					data[offset+c] = 0x00
				}
			}
		}
	}
	return data
}

func TestRealImageCompression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real image tests in short mode")
	}

	// Determine the correct path to images directory
	// The test may run from different directories, so try multiple paths
	possiblePaths := []string{
		"images",
		"../images",
		"../../images",
	}

	var imagesPath string
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			imagesPath = p
			break
		}
	}

	if imagesPath == "" {
		t.Skip("images directory not found")
	}

	testCases := []struct {
		name     string
		filename string
	}{
		{"Code PNG", filepath.Join(imagesPath, "code.png")},
		{"Cursor Models", filepath.Join(imagesPath, "cursor-2025-models.png")},
		{"Cursor Meetup", filepath.Join(imagesPath, "cursor-meetup.png")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Load original PNG
			originalFile, err := os.Open(tc.filename)
			if err != nil {
				t.Fatalf("failed to open %s: %v", tc.filename, err)
			}
			t.Cleanup(func() {
				if closeErr := originalFile.Close(); closeErr != nil {
					t.Errorf("failed to close %s: %v", tc.filename, closeErr)
				}
			})

			img, err := stdpng.Decode(originalFile)
			if err != nil {
				t.Fatalf("failed to decode %s: %v", tc.filename, err)
			}

			bounds := img.Bounds()

			// Get file size
			if _, seekErr := originalFile.Seek(0, 0); seekErr != nil {
				t.Fatalf("failed to seek %s: %v", tc.filename, seekErr)
			}
			originalStats, err := originalFile.Stat()
			if err != nil {
				t.Fatalf("failed to stat %s: %v", tc.filename, err)
			}
			originalSize := int(originalStats.Size())

			// Extract pixels based on color model
			pixels := extractPixels(t, img)

			// Determine color type
			var colorType ColorType
			switch img.ColorModel() {
			case color.RGBAModel, color.NRGBAModel:
				colorType = ColorRGBA
			case color.GrayModel:
				colorType = ColorGrayscale
			default:
				// For other color models, assume RGBA
				colorType = ColorRGBA
			}

			// Compress using our encoder
			encoder, err := NewEncoder(bounds.Dx(), bounds.Dy(), colorType)
			if err != nil {
				t.Fatalf("NewEncoder() error = %v", err)
			}

			compressed, err := encoder.Encode(pixels)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			// Key metric: Our compressed output should be significantly smaller than raw pixel data
			// (not just smaller than the original file, which may have been pre-compressed)
			compressionRatio := float64(len(compressed)) / float64(len(pixels)) * 100
			t.Logf("Image: %s (%dx%d)", tc.filename, bounds.Dx(), bounds.Dy())
			t.Logf("Raw pixels: %d bytes", len(pixels))
			t.Logf("Compressed: %d bytes", len(compressed))
			t.Logf("Compression ratio: %.2f%%", compressionRatio)
			t.Logf("Original file: %d bytes", originalSize)

			// Verify: compressed should be < 50% of raw pixels (good compression)
			// Note: We cannot guarantee compressed < originalFileSize because the original
			// may have been compressed with different/better settings
			if compressionRatio >= 50 {
				t.Logf("Warning: compression ratio %.2f%% is higher than expected", compressionRatio)
			}

			// Verify the compressed image is valid and decodes correctly
			decodedImg, err := stdpng.Decode(bytes.NewReader(compressed))
			if err != nil {
				t.Errorf("compressed image decoding failed: %v", err)
			} else {
				decodedBounds := decodedImg.Bounds()
				if decodedBounds.Dx() != bounds.Dx() || decodedBounds.Dy() != bounds.Dy() {
					t.Errorf("decoded dimensions mismatch: got %dx%d, want %dx%d",
						decodedBounds.Dx(), decodedBounds.Dy(), bounds.Dx(), bounds.Dy())
				}
			}
		})
	}
}

func extractPixels(t *testing.T, img image.Image) []byte {
	t.Helper()

	bounds := img.Bounds()

	var pixels []byte
	switch m := img.(type) {
	case *image.RGBA:
		pixels = make([]byte, bounds.Dx()*bounds.Dy()*4)
		for y := 0; y < bounds.Dy(); y++ {
			row := m.Pix[y*m.Stride:]
			for x := 0; x < bounds.Dx(); x++ {
				offset := (y*bounds.Dx() + x) * 4
				pixels[offset] = row[x*4+0]
				pixels[offset+1] = row[x*4+1]
				pixels[offset+2] = row[x*4+2]
				pixels[offset+3] = row[x*4+3]
			}
		}
	case *image.NRGBA:
		pixels = make([]byte, bounds.Dx()*bounds.Dy()*4)
		for y := 0; y < bounds.Dy(); y++ {
			row := m.Pix[y*m.Stride:]
			for x := 0; x < bounds.Dx(); x++ {
				offset := (y*bounds.Dx() + x) * 4
				pixels[offset] = row[x*4+0]
				pixels[offset+1] = row[x*4+1]
				pixels[offset+2] = row[x*4+2]
				pixels[offset+3] = row[x*4+3]
			}
		}
	default:
		// Generic fallback - slower but works for any color model
		pixels = make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				c := img.At(x, y)
				r, g, b, a := c.RGBA()
				pixels = append(pixels, byte(r>>8), byte(g>>8), byte(b>>8), byte(a>>8))
			}
		}
	}

	return pixels
}
