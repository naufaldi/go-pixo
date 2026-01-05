package png

import (
	"bytes"
	"fmt"
	"io"
)

type Encoder struct {
	width     int
	height    int
	colorType ColorType
	opts      Options
}

func NewEncoder(width, height int, colorType ColorType) (*Encoder, error) {
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidDimensions
	}

	// Validate parameters by creating a dummy IHDR
	if _, err := NewIHDRData(width, height, 8, uint8(colorType)); err != nil {
		return nil, err
	}

	opts := FastOptions(width, height)
	opts.ColorType = colorType

	return &Encoder{
		width:     width,
		height:    height,
		colorType: colorType,
		opts:      opts,
	}, nil
}

func NewEncoderWithOptions(opts Options) (*Encoder, error) {
	if opts.Width <= 0 || opts.Height <= 0 {
		return nil, ErrInvalidDimensions
	}

	// Validate parameters by creating a dummy IHDR
	if _, err := NewIHDRData(opts.Width, opts.Height, 8, uint8(opts.ColorType)); err != nil {
		return nil, err
	}

	return &Encoder{
		width:     opts.Width,
		height:    opts.Height,
		colorType: opts.ColorType,
		opts:      opts,
	}, nil
}

func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	return e.EncodeWithOptions(pixels, e.opts)
}

func (e *Encoder) EncodeWithOptions(pixels []byte, opts Options) ([]byte, error) {
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("preprocess", 0)
	}

	colorType := opts.ColorType
	bpp := BytesPerPixel(colorType)
	expectedSize := opts.Width * opts.Height * bpp
	if len(pixels) != expectedSize {
		return nil, fmt.Errorf("png: pixel count mismatch: got %d bytes, want %d", len(pixels), expectedSize)
	}

	processedPixels := pixels

	// 0. Quantization (Lossy) - before other optimizations
	if opts.MaxColors > 0 && opts.MaxColors < 256 {
		var indexedPixels []byte
		var palette Palette

		if opts.Dithering {
			indexedPixels, palette = QuantizeWithDithering(processedPixels, int(colorType), opts.MaxColors)
		} else {
			indexedPixels, palette = Quantize(processedPixels, int(colorType), opts.MaxColors)
		}

		if opts.ProgressCallback != nil {
			opts.ProgressCallback("preprocess", 100)
		}

		var buf bytes.Buffer

		if err := writeSignature(&buf); err != nil {
			return nil, err
		}

		if err := writeIHDR(&buf, opts.Width, opts.Height, ColorIndexed); err != nil {
			return nil, err
		}

		if err := writePreservedChunks(&buf, opts.PreserveChunks); err != nil {
			return nil, err
		}

		if err := WritePLTE(&buf, palette); err != nil {
			return nil, err
		}

		if err := WriteIDATWithOptions(&buf, indexedPixels, opts.Width, opts.Height, ColorIndexed, opts); err != nil {
			return nil, err
		}

		if err := writeIEND(&buf); err != nil {
			return nil, err
		}

		result := buf.Bytes()

		// Size guarantee: if compressed is larger than original, return minimal PNG
		if opts.EnsureSizeNotLarger && len(result) >= originalSizeForGuarantee(opts, pixels) {
			return e.encodeMinimalPNG(pixels, opts.Width, opts.Height, opts.ColorType)
		}

		return result, nil
	}

	// 1. Color Reduction (Lossless)
	if opts.ReduceColorType {
		if CanReduceToRGB(processedPixels, opts.Width, opts.Height) {
			var err error
			processedPixels, colorType, err = ReduceToRGB(processedPixels, opts.Width, opts.Height)
			if err != nil {
				return nil, err
			}
		} else if CanReduceToGrayscale(processedPixels, opts.Width, opts.Height, colorType) {
			var err error
			processedPixels, colorType, err = ReduceToGrayscale(processedPixels, opts.Width, opts.Height, colorType)
			if err != nil {
				return nil, err
			}
		}
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("preprocess", 100)
		opts.ProgressCallback("filtering", 0)
	}

	// 2. Alpha Optimization (RGB=0 when A=0)
	if opts.OptimizeAlpha && colorType == ColorRGBA {
		processedPixels = OptimizeAlpha(processedPixels, colorType)
	}

	var buf bytes.Buffer

	// 3. Write PNG Signature
	if err := writeSignature(&buf); err != nil {
		return nil, err
	}

	// 4. Write IHDR Chunk (Critical)
	if err := writeIHDR(&buf, opts.Width, opts.Height, colorType); err != nil {
		return nil, err
	}

	// Optional ancillary chunk passthrough (e.g. color profile chunks for visual fidelity)
	if err := writePreservedChunks(&buf, opts.PreserveChunks); err != nil {
		return nil, err
	}

	// 5. Write IDAT Chunk (Critical) - Includes Filter Strategy and Deflate Compression
	if err := WriteIDATWithOptions(&buf, processedPixels, opts.Width, opts.Height, colorType, opts); err != nil {
		return nil, err
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("deflate", 100)
		opts.ProgressCallback("finalize", 0)
	}

	// 6. Write IEND Chunk (Critical)
	if err := writeIEND(&buf); err != nil {
		return nil, err
	}

	if opts.ProgressCallback != nil {
		opts.ProgressCallback("finalize", 100)
	}

	result := buf.Bytes()

	// Size guarantee: if compressed is larger than original, return minimal PNG
	if opts.EnsureSizeNotLarger && len(result) >= originalSizeForGuarantee(opts, pixels) {
		return e.encodeMinimalPNG(pixels, opts.Width, opts.Height, opts.ColorType)
	}

	return result, nil
}

// encodeMinimalPNG creates a minimal PNG when compression doesn't help.
// Uses stored blocks for fast encoding without DEFLATE overhead.
func (e *Encoder) encodeMinimalPNG(pixels []byte, width, height int, colorType ColorType) ([]byte, error) {
	var buf bytes.Buffer

	if err := writeSignature(&buf); err != nil {
		return nil, err
	}

	if err := writeIHDR(&buf, width, height, colorType); err != nil {
		return nil, err
	}

	// Use stored blocks for minimal compression (fast, no expansion for already-compressed data)
	// We call WriteIDATWithOptions with a special preset/level if needed,
	// but the default behavior with CompressionLevel=0 or if it's smaller will use stored blocks.
	opts := e.opts
	opts.Width = width
	opts.Height = height
	opts.ColorType = colorType
	// Force stored blocks by using level 0
	opts.CompressionLevel = 0

	if err := WriteIDATWithOptions(&buf, pixels, width, height, colorType, opts); err != nil {
		return nil, err
	}

	if err := writeIEND(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func writeSignature(w io.Writer) error {
	_, err := w.Write(Signature())
	return err
}

func writeIHDR(w io.Writer, width, height int, colorType ColorType) error {
	ihdr, err := NewIHDRData(width, height, 8, uint8(colorType))
	if err != nil {
		return err
	}

	return WriteIHDR(w, ihdr)
}

func writeIEND(w io.Writer) error {
	return WriteIEND(w)
}

func writePreservedChunks(w io.Writer, chunks []Chunk) error {
	for _, c := range chunks {
		if c.IsRequired() {
			continue
		}
		if c.chunkType == ChunkIDAT {
			continue
		}
		if _, err := c.WriteTo(w); err != nil {
			return err
		}
	}
	return nil
}

func originalSizeForGuarantee(opts Options, pixels []byte) int {
	if opts.OriginalFileSize > 0 {
		return opts.OriginalFileSize
	}
	return len(pixels)
}
