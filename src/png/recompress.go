package png

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	stdpng "image/png"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// RecompressPNGBytesLossless recompresses an input PNG while preserving pixels.
//
// It returns the smallest valid output between:
//   - the original input bytes (guarantee: never larger), and
//   - a newly-encoded PNG using go-pixo's encoder.
//
// opts acts as a compression template; dimensions and color type come from the input.
// Quantization, dithering, and invisible RGB changes are disabled. Unsupported
// 16-bit images are returned unchanged because the encoder accepts only 8-bit samples.
func RecompressPNGBytesLossless(inputPNG []byte, opts Options) ([]byte, error) {
	if err := imagelimit.CheckEncodedSize(len(inputPNG)); err != nil {
		return nil, err
	}
	config, err := stdpng.DecodeConfig(bytes.NewReader(inputPNG))
	if err != nil {
		return nil, fmt.Errorf("png: decode config: %w", err)
	}
	if err := imagelimit.CheckDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}

	preserved, err := ExtractPreserveChunks(inputPNG)
	if err != nil {
		return nil, err
	}

	img, err := stdpng.Decode(bytes.NewReader(inputPNG))
	if err != nil {
		return nil, fmt.Errorf("png: decode input: %w", err)
	}

	switch img.(type) {
	case *image.Gray16, *image.RGBA64, *image.NRGBA64:
		return inputPNG, nil
	}

	b := img.Bounds()
	width := b.Dx()
	height := b.Dy()
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidDimensions
	}

	nrgba := image.NewNRGBA(b)
	if paletted, ok := img.(*image.Paletted); ok {
		// draw.Draw premultiplies palette colors, losing RGB when alpha is zero.
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				nrgba.Set(x, y, paletted.At(x, y))
			}
		}
	} else {
		draw.Draw(nrgba, b, img, b.Min, draw.Src)
	}

	encOpts := opts
	encOpts.ApplyLossless()
	encOpts.OptimizeAlpha = false
	encOpts.Width = width
	encOpts.Height = height
	encOpts.ColorType = ColorRGBA
	encOpts.PreserveChunks = preserved
	// Compare the candidate directly against the original below; do not encode a
	// second raw-pixel fallback when the original bytes are already available.
	encOpts.EnsureSizeNotLarger = false
	encOpts.OriginalFileSize = len(inputPNG)

	encoder, err := NewEncoderWithOptions(encOpts)
	if err != nil {
		return nil, err
	}

	out, err := encoder.EncodeWithOptions(nrgba.Pix, encOpts)
	if err != nil {
		return nil, err
	}

	if _, err := stdpng.Decode(bytes.NewReader(out)); err != nil {
		return nil, fmt.Errorf("png: output decode failed: %w", err)
	}

	if len(out) >= len(inputPNG) {
		return inputPNG, nil
	}
	return out, nil
}
