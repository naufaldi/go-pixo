package png

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/mac/go-pixo/src/compress"
)

// WriteIDAT writes a complete IDAT chunk containing the compressed image data.
// It writes:
//   - zlib header (CMF + FLG bytes)
//   - DEFLATE-compressed data (fixed or dynamic Huffman blocks)
//   - zlib footer (Adler32 checksum)
//   - wrapped in an IDAT chunk (length + "IDAT" + data + CRC)
func WriteIDAT(w interface{ Write([]byte) (int, error) }, pixels []byte, width, height int, colorType ColorType) error {
	opts := BalancedOptions(width, height)
	opts.ColorType = colorType
	return WriteIDATWithOptions(w, pixels, width, height, colorType, opts)
}

// WriteIDATWithOptions writes IDAT chunk with configurable options.
func WriteIDATWithOptions(w interface{ Write([]byte) (int, error) }, pixels []byte, width, height int, colorType ColorType, opts Options) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidDimensions
	}

	bpp := BytesPerPixel(colorType)
	expectedRawLen := width * bpp * height

	if len(pixels) != expectedRawLen {
		return fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	// Build scanlines with filter selection based on strategy
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		// Report progress periodically for filtering
		if opts.ProgressCallback != nil && y%100 == 0 {
			percent := int(float64(y) / float64(height) * 100)
			opts.ProgressCallback("filtering", percent)
		}

		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, filteredRow := SelectFilterWithStrategy(row, prevRow, bpp, opts.FilterStrategy)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("filtering", 100)
		opts.ProgressCallback("deflate", 0)
	}

	// Build zlib-compressed data
	zlibData, err := buildZlibData(scanlineData, opts)
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
// This function uses size comparison fallback: if DEFLATE doesn't reduce the size,
// it falls back to stored blocks (uncompressed) to ensure the output is never larger.
// When ZopfliEnabled is true, uses Zopfli iterative compression for maximum compression.
func buildZlibData(pixels []byte, opts Options) ([]byte, error) {
	// Write zlib header: CMF (DEFLATE, 32K window) + FLG (default compression, check bits)
	cmf, err := compress.ZlibHeaderBytes(32768, 2)
	if err != nil {
		return nil, err
	}

	// Compress scanline data using DEFLATE with size comparison fallback
	// This ensures the output is never larger than the input
	encoder := compress.NewDeflateEncoder()
	encoder.SetCompressionLevel(opts.CompressionLevel)

	var deflateData []byte

	if opts.ZopfliEnabled {
		deflateData, err = buildZlibDataWithZopfli(pixels, opts)
		if err != nil {
			log.Printf("png: Zopfli compression failed, falling back to standard DEFLATE: %v", err)
			deflateData, err = encoder.EncodeAuto(pixels)
			if err != nil {
				return nil, fmt.Errorf("failed to compress scanline data: %w", err)
			}
		}
	} else if opts.OptimalDeflate {
		// Use optimal DEFLATE parsing for better compression
		config := opts.OptimalConfig
		if config.MaxIterations <= 0 {
			config = compress.OptimalConfigForLevel(opts.CompressionLevel)
		}

		// Set up progress callback if provided
		if opts.ProgressCallback != nil {
			config.ProgressCallback = func(iteration, improvement float64) {
				percent := int(iteration / float64(config.MaxIterations) * 100)
				if percent > 100 {
					percent = 100
				}
				opts.ProgressCallback("deflate", percent)
			}
		}

		// Use optimal parsing for better compression ratios (3-8% improvement)
		tokens, err := compress.OptimalParse(pixels, config)
		if err != nil {
			// Fallback to standard encoding
			deflateData, err = encoder.EncodeAuto(pixels)
			if err != nil {
				return nil, fmt.Errorf("failed to compress scanline data: %w", err)
			}
		} else {
			// Encode tokens to DEFLATE format
			var buf bytes.Buffer
			if err := compress.WriteDynamicBlock(&buf, true, tokens); err != nil {
				// Fallback to standard encoding
				deflateData, err = encoder.EncodeAuto(pixels)
				if err != nil {
					return nil, fmt.Errorf("failed to compress scanline data: %w", err)
				}
			} else {
				deflateData = buf.Bytes()
			}
		}
	} else {
		// Use fallback: if DEFLATE doesn't help, use stored blocks
		deflateData, err = encoder.EncodeWithFallback(pixels)
		if err != nil {
			return nil, fmt.Errorf("failed to compress scanline data: %w", err)
		}
	}

	// Write Adler32 checksum of the uncompressed scanline data
	adler := compress.Adler32(pixels)
	adlerBuf := compress.ZlibFooterBytes(adler)

	// Combine: zlib header + DEFLATE/stored block data + Adler32 footer
	result := make([]byte, 0, len(cmf)+len(deflateData)+len(adlerBuf))
	result = append(result, cmf...)
	result = append(result, deflateData...)
	result = append(result, adlerBuf[:]...)

	// Optionally compare against stdlib zlib output and keep the smaller stream.
	// This is still "standard Go" (no third-party deps) and helps keep results
	// competitive for already-compressed PNGs.
	stdlibZlib, stdlibErr := compress.ZlibCompressStdlib(pixels, opts.CompressionLevel)
	if stdlibErr == nil && len(stdlibZlib) > 0 && len(stdlibZlib) < len(result) {
		return stdlibZlib, nil
	}

	return result, nil
}

// buildZlibDataWithZopfli compresses data using Zopfli iterative compression.
// It uses the Options.ZopfliIterations, ZopfliBlockSplitting, and ZopfliSplitThreshold settings.
// A progress callback can be provided to track iteration progress.
func buildZlibDataWithZopfli(pixels []byte, opts Options) ([]byte, error) {
	if len(pixels) == 0 {
		return []byte{}, nil
	}

	iterations := opts.ZopfliIterations
	if iterations <= 0 {
		iterations = compress.DefaultZopfliIterations
	}

	config := compress.NewZopfliIterationConfig()
	config.Iterations = iterations
	config.BlockSplitting = opts.ZopfliBlockSplitting
	config.SplitThreshold = opts.ZopfliSplitThreshold

	if opts.ProgressCallback != nil {
		config.ProgressCallback = func(iteration, improvement float64, size int) {
			percent := int(float64(iteration) / float64(iterations) * 100)
			if percent > 100 {
				percent = 100
			}
			opts.ProgressCallback("deflate", percent)
		}
	}

	startTime := time.Now()
	result, err := compress.ZopfliIterate(pixels, config)
	elapsed := time.Since(startTime)

	if elapsed > 5*time.Second {
		log.Printf("png: Zopfli iteration took %v (%d iterations)", elapsed, iterations)
	}

	if err != nil {
		return nil, fmt.Errorf("Zopfli compression failed: %w", err)
	}

	return result, nil
}

// IDATDataBytes returns the raw zlib data for IDAT without the chunk wrapper.
// This is useful for testing or when you need to write multiple IDAT chunks.
func IDATDataBytes(pixels []byte, width, height int, colorType ColorType) ([]byte, error) {
	opts := BalancedOptions(width, height)
	opts.ColorType = colorType
	return IDATDataBytesWithOptions(pixels, width, height, colorType, opts)
}

// IDATDataBytesWithOptions returns the raw zlib data with configurable options.
func IDATDataBytesWithOptions(pixels []byte, width, height int, colorType ColorType, opts Options) ([]byte, error) {
	bpp := BytesPerPixel(colorType)
	expectedRawLen := width * bpp * height

	if len(pixels) != expectedRawLen {
		return nil, fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	// Build scanlines with filter selection based on strategy
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	var prevRow []byte
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, filteredRow := SelectFilterWithStrategy(row, prevRow, bpp, opts.FilterStrategy)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
	}

	return buildZlibData(scanlineData, opts)
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
