package png

import (
	"fmt"

	"github.com/mac/go-pixo/src/compress"
)

// WriteIDAT writes a complete IDAT chunk containing the compressed image data.
// It writes:
//   - zlib header (CMF + FLG bytes)
//   - DEFLATE-compressed data (fixed or dynamic Huffman blocks)
//   - zlib footer (Adler32 checksum)
//   - wrapped in an IDAT chunk (length + "IDAT" + data + CRC)
func WriteIDAT(w interface{ Write([]byte) (int, error) }, pixels []byte, width, height int, colorType ColorType) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidDimensions
	}

	bpp := BytesPerPixel(colorType)
	expectedRawLen := width * bpp * height

	if len(pixels) != expectedRawLen {
		return fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	// Build scanlines with per-row filter selection
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, filteredRow := SelectFilter(row, prevRow, bpp)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	// Build zlib-compressed data
	zlibData, err := buildZlibData(scanlineData, width, height, colorType)
	if err != nil {
		return fmt.Errorf("png: failed to build zlib data: %w", err)
	}

	// Write as IDAT chunk
	chunk := Chunk{
		chunkType: ChunkIDAT,
		Data:      zlibData,
	}
	_, err = chunk.WriteTo(w)
	return err
}

// buildZlibData builds the zlib-wrapped DEFLATE data containing scanlines.
// The pixels parameter contains all scanline data with filter bytes prepended.
func buildZlibData(pixels []byte, width, height int, colorType ColorType) ([]byte, error) {
	// Write zlib header: CMF (DEFLATE, 32K window) + FLG (default compression, check bits)
	cmf, err := compress.ZlibHeaderBytes(32768, 2)
	if err != nil {
		return nil, err
	}

	// Compress scanline data using DEFLATE with auto table selection
	// EncodeAuto tries both fixed and dynamic tables and picks the smaller
	encoder := compress.NewDeflateEncoder()
	deflateData, err := encoder.EncodeAuto(pixels)
	if err != nil {
		return nil, fmt.Errorf("failed to compress scanline data: %w", err)
	}

	// Write Adler32 checksum of the uncompressed scanline data
	adler := compress.Adler32(pixels)
	adlerBuf := compress.ZlibFooterBytes(adler)

	// Combine: zlib header + DEFLATE data + Adler32 footer
	result := make([]byte, 0, len(cmf)+len(deflateData)+len(adlerBuf))
	result = append(result, cmf...)
	result = append(result, deflateData...)
	result = append(result, adlerBuf[:]...)

	return result, nil
}

// IDATDataBytes returns the raw zlib data for IDAT without the chunk wrapper.
// This is useful for testing or when you need to write multiple IDAT chunks.
func IDATDataBytes(pixels []byte, width, height int, colorType ColorType) ([]byte, error) {
	bpp := BytesPerPixel(colorType)
	expectedRawLen := width * bpp * height

	if len(pixels) != expectedRawLen {
		return nil, fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	// Build scanlines with per-row filter selection
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, filteredRow := SelectFilter(row, prevRow, bpp)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	return buildZlibData(scanlineData, width, height, colorType)
}

// ExpectedIDATSize returns an estimated size of the IDAT chunk data for a given image.
// The actual size may vary due to DEFLATE compression, so this is only an approximation.
func ExpectedIDATSize(width, height int, colorType ColorType) int {
	bpp := BytesPerPixel(colorType)
	scanlineLen := 1 + width*bpp
	uncompressedSize := scanlineLen * height
	// Estimate: zlib header (2) + compressed data (assume 50% compression) + Adler32 (4)
	// This is a rough estimate; actual compression ratio depends on image content
	estimatedCompressed := uncompressedSize / 2
	if estimatedCompressed < 10 {
		estimatedCompressed = 10
	}
	return 2 + estimatedCompressed + 4
}
