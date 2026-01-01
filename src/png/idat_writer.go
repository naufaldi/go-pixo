package png

import (
	"fmt"

	"github.com/mac/go-pixo/src/compress"
)

// WriteIDAT writes a complete IDAT chunk containing the compressed image data.
// It writes:
//   - zlib header (CMF + FLG bytes)
//   - stored block(s) containing filter bytes + pixel scanlines
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

	// Build scanlines with filter byte 0 (None) prepended to each row
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		scanlineData = append(scanlineData, 0)
		scanlineData = append(scanlineData, pixels[offset:offset+width*bpp]...)
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
func buildZlibData(pixels []byte, width, height int, colorType ColorType) ([]byte, error) {
	bpp := BytesPerPixel(colorType)
	scanlineLen := 1 + width*bpp

	// Estimate buffer size: zlib header (2) + max stored blocks + adler32 (4)
	// Each scanline: 1 filter byte + width*bpp pixels
	// Each stored block: 1 header + 4 footer + data
	estimatedSize := 2 + (1+4+scanlineLen)*height + 4
	buf := make([]byte, 0, estimatedSize)

	// Write zlib header: CMF (DEFLATE, 32K window) + FLG (default compression, check bits)
	cmf, err := compress.ZlibHeaderBytes(32768, 2)
	if err != nil {
		return nil, err
	}
	buf = append(buf, cmf[:]...)

	// Write scanlines wrapped in stored blocks
	// For Phase 1, we use filter type 0 (None) for simplicity
	for y := 0; y < height; y++ {
		offset := y * (1 + width*bpp)
		scanlineData := pixels[offset : offset+1+width*bpp]

		// Each scanline goes in its own stored block (final block for last scanline)
		isFinal := (y == height-1)

		// Build the stored block
		// Header (1 byte) + LEN (2 bytes) + NLEN (2 bytes) + data
		blockData := make([]byte, 1+4+len(scanlineData))

		// Header: type=000, BFINAL
		if isFinal {
			blockData[0] = 0x01 // Final block
		} else {
			blockData[0] = 0x00 // Not final
		}

		// LEN: little-endian length of data
		dataLen := uint16(len(scanlineData))
		blockData[1] = byte(dataLen)
		blockData[2] = byte(dataLen >> 8)

		// NLEN: one's complement of LEN
		nlen := ^dataLen
		blockData[3] = byte(nlen)
		blockData[4] = byte(nlen >> 8)

		// Copy scanline data (filter byte + pixels)
		copy(blockData[5:], scanlineData)

		buf = append(buf, blockData...)
	}

	// Write Adler32 checksum of the uncompressed scanline data
	adler := compress.Adler32(pixels)
	adlerBuf := compress.ZlibFooterBytes(adler)
	buf = append(buf, adlerBuf[:]...)

	return buf, nil
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

	// Build scanlines with filter byte 0 (None) prepended to each row
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		scanlineData = append(scanlineData, 0)
		scanlineData = append(scanlineData, pixels[offset:offset+width*bpp]...)
	}

	return buildZlibData(scanlineData, width, height, colorType)
}

// ExpectedIDATSize returns the expected size of the IDAT chunk data for a given image.
func ExpectedIDATSize(width, height int, colorType ColorType) int {
	bpp := BytesPerPixel(colorType)
	scanlineLen := 1 + width*bpp
	// zlib header (2) + scanlines in stored blocks (each: 1 header + 4 footer + scanline) + Adler32 (4)
	// = 2 + height * (5 + scanlineLen) + 4
	return 2 + height*(5+scanlineLen) + 4
}
