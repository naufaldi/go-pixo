package png

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// Encoder writes PNG images with configurable compression and preprocessing options.
type Encoder struct {
	width     int
	height    int
	colorType ColorType
	opts      Options
}

// NewEncoder creates a PNG encoder for an image of the given size and color type.
func NewEncoder(width, height int, colorType ColorType) (*Encoder, error) {
	if _, err := pixelDataSize(width, height, colorType); err != nil {
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

// NewEncoderWithOptions creates a PNG encoder using the provided Options.
func NewEncoderWithOptions(opts Options) (*Encoder, error) {
	if _, err := pixelDataSize(opts.Width, opts.Height, opts.ColorType); err != nil {
		return nil, err
	}

	return &Encoder{
		width:     opts.Width,
		height:    opts.Height,
		colorType: opts.ColorType,
		opts:      opts,
	}, nil
}

// Encode encodes pixels into a PNG using the encoder's default options.
func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	return e.EncodeWithOptions(pixels, e.opts)
}

// EncodeWithOptions encodes pixels into a PNG using opts.
func (e *Encoder) EncodeWithOptions(pixels []byte, opts Options) ([]byte, error) {
	expectedSize, err := pixelDataSize(opts.Width, opts.Height, opts.ColorType)
	if err != nil {
		return nil, err
	}
	if len(pixels) != expectedSize {
		return nil, fmt.Errorf("png: pixel count mismatch: got %d bytes, want %d", len(pixels), expectedSize)
	}
	if err := validateFilterOptions(opts); err != nil {
		return nil, err
	}
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("preprocess", 0)
	}

	colorType := opts.ColorType

	processedPixels := pixels

	// 0. Quantization (Lossy) - before other optimizations
	if opts.MaxColors > 0 && opts.MaxColors <= 256 {
		// Palette stores RGB only. Do not silently discard alpha or pass grayscale
		// samples into the RGB quantizer. Lossless encoding supports both inputs.
		if colorType != ColorRGB && colorType != ColorRGBA {
			return nil, fmt.Errorf("png: palette quantization requires RGB or opaque RGBA pixels")
		}
		if colorType == ColorRGBA {
			for i := 3; i < len(pixels); i += 4 {
				if pixels[i] != 255 {
					return nil, fmt.Errorf("png: palette quantization does not support transparency; use lossless encoding")
				}
			}
		}
		// Avoid the legacy full RGB lookup table here. Linear nearest-color search
		// is exact, bounded by 256 palette entries, and needs no image-sized scratch.
		indexedPixels, palette := QuantizeWithOptions(processedPixels, int(colorType), opts.MaxColors, opts, false)

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

		// Raw-pixel callers can request a stored fallback at their size target.
		if opts.EnsureSizeNotLarger && len(result) >= originalSizeForGuarantee(opts, pixels) {
			return e.encodeMinimalPNG(pixels, opts)
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

	// Raw-pixel callers can request a stored fallback at their size target.
	if opts.EnsureSizeNotLarger && len(result) >= originalSizeForGuarantee(opts, pixels) {
		return e.encodeMinimalPNG(pixels, opts)
	}

	return result, nil
}

func pixelDataSize(width, height int, colorType ColorType) (int, error) {
	if _, err := NewIHDRData(width, height, 8, uint8(colorType)); err != nil {
		return 0, err
	}
	return imagelimit.PixelBytes(width, height, BytesPerPixel(colorType))
}

// encodeMinimalPNG writes an unfiltered PNG using stored blocks and preserved
// metadata. Without the original file bytes it cannot guarantee a smaller file.
func (e *Encoder) encodeMinimalPNG(pixels []byte, opts Options) ([]byte, error) {
	var buf bytes.Buffer

	if err := writeSignature(&buf); err != nil {
		return nil, err
	}

	if err := writeIHDR(&buf, opts.Width, opts.Height, opts.ColorType); err != nil {
		return nil, err
	}

	if err := writePreservedChunks(&buf, opts.PreserveChunks); err != nil {
		return nil, err
	}

	// Use this call's metadata and skip filter search as well as LZ77 matching.
	opts.CompressionLevel = 0
	opts.OptimalDeflate = false
	opts.FilterStrategy = FilterStrategyNone
	opts.FilterEarlyTermination = false

	if err := WriteIDATWithOptions(&buf, pixels, opts.Width, opts.Height, opts.ColorType, opts); err != nil {
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
