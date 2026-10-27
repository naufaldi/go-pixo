package png

import (
	"bytes"
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
	opts := BalancedOptions(width, height)
	opts.ColorType = colorType
	return WriteIDATWithOptions(w, pixels, width, height, colorType, opts)
}

// WriteIDATWithOptions writes IDAT chunk with configurable options.
func WriteIDATWithOptions(w interface{ Write([]byte) (int, error) }, pixels []byte, width, height int, colorType ColorType, opts Options) error {
	expectedRawLen, err := pixelDataSize(width, height, colorType)
	if err != nil {
		return err
	}
	if err := validateFilterOptions(opts); err != nil {
		return err
	}

	bpp := BytesPerPixel(colorType)

	if len(pixels) != expectedRawLen {
		return fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	scanlineData := filterScanlines(pixels, width, height, bpp, opts)

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
// Level zero writes stored blocks directly. Other non-optimal levels fall back
// to stored blocks when compression does not help. PNG/zlib framing adds overhead;
// only the byte-recompression wrapper can guarantee never-larger files.
func buildZlibData(pixels []byte, opts Options) ([]byte, error) {
	// Write zlib header: CMF (DEFLATE, 32K window) + FLG (default compression, check bits)
	cmf, err := compress.ZlibHeaderBytes(32768, 2)
	if err != nil {
		return nil, err
	}

	var deflateData []byte
	if opts.CompressionLevel == 0 {
		// Bypass both LZ77 allocation and optimal search, even when the caller's
		// template has OptimalDeflate enabled. Stored blocks split at 65535 bytes.
		var stored bytes.Buffer
		err = compress.WriteStoredBlocks(&stored, pixels)
		deflateData = stored.Bytes()
	} else {
		encoder := compress.NewDeflateEncoder()
		encoder.SetCompressionLevel(opts.CompressionLevel)
		if opts.ProgressCallback != nil {
			encoder.ProgressCallback = func(iteration, total int) {
				percent := int(float64(iteration) / float64(total) * 100)
				opts.ProgressCallback("deflate", percent)
			}
		}
		if opts.OptimalDeflate {
			iterations := opts.ZopfliIterations
			if opts.OptimalConfig.MaxIterations > 0 {
				iterations = opts.OptimalConfig.MaxIterations
			}
			deflateData, err = encoder.EncodeOptimalWithConfig(pixels, iterations, opts.OptimalConfig.BlockSplitting)
		} else {
			deflateData, err = encoder.EncodeWithFallback(pixels)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to compress scanline data: %w", err)
	}

	// Write Adler32 checksum of the uncompressed scanline data
	adler := compress.Adler32(pixels)
	adlerBuf := compress.ZlibFooterBytes(adler)

	// Combine: zlib header + DEFLATE/stored block data + Adler32 footer
	result := make([]byte, 0, len(cmf)+len(deflateData)+len(adlerBuf))
	result = append(result, cmf...)
	result = append(result, deflateData...)
	result = append(result, adlerBuf[:]...)

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
	expectedRawLen, err := pixelDataSize(width, height, colorType)
	if err != nil {
		return nil, err
	}
	if err := validateFilterOptions(opts); err != nil {
		return nil, err
	}
	bpp := BytesPerPixel(colorType)

	if len(pixels) != expectedRawLen {
		return nil, fmt.Errorf("png: pixel data length %d does not match expected %d for %dx%d image",
			len(pixels), expectedRawLen, width, height)
	}

	return buildZlibData(filterScanlines(pixels, width, height, bpp, opts), opts)
}

// filterScanlines owns scratch for one encode. Each winner is copied into the
// scanline stream before the next row can overwrite its buffer.
func filterScanlines(pixels []byte, width, height, bpp int, opts Options) []byte {
	scanlineData := make([]byte, 0, (1+width*bpp)*height)
	scratch := NewAdaptiveScratchWithConfig(width*bpp, bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:   opts.FilterEarlyTermination,
		FilterSelectionThreshold:  float64(width * bpp),
		EarlyTerminationThreshold: 0.5 * float64(width*bpp),
		MinFiltersToTry:           DefaultMinFiltersToTry,
	})
	progress := newWorkProgress(opts.ProgressCallback, "filtering", height, 1)
	var prevRow []byte
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, filteredRow := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, opts.FilterStrategy, scratch)
		scanlineData = append(scanlineData, byte(filterType))
		scanlineData = append(scanlineData, filteredRow...)
		prevRow = row
		progress.report(y + 1)
	}
	progress.finish()
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("deflate", 0)
	}
	return scanlineData
}

func validateFilterOptions(opts Options) error {
	if opts.FilterEarlyTermination {
		switch opts.FilterStrategy {
		case FilterStrategyMinSum, FilterStrategyAdaptive, FilterStrategyParallel, FilterStrategyEntropy:
		default:
			return fmt.Errorf("png: early filter termination requires MinSum, Adaptive, Parallel, or Entropy strategy")
		}
	}
	return nil
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
