package png

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	stdpng "image/png"
)

// RecompressPNGBytesLossless recompresses an input PNG while preserving pixels.
//
// It returns the smallest valid output between:
//   - the original input bytes (guarantee: never larger), and
//   - a newly-encoded PNG using go-pixo's encoder.
//
// opts acts as a template; Width/Height/ColorType are overridden to match the decoded image.
func RecompressPNGBytesLossless(inputPNG []byte, opts Options) ([]byte, error) {
	preserved, err := ExtractPreserveChunks(inputPNG)
	if err != nil {
		return nil, err
	}

	img, err := stdpng.Decode(bytes.NewReader(inputPNG))
	if err != nil {
		return nil, fmt.Errorf("png: decode input: %w", err)
	}

	b := img.Bounds()
	width := b.Dx()
	height := b.Dy()
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidDimensions
	}

	nrgba := image.NewNRGBA(b)
	draw.Draw(nrgba, b, img, b.Min, draw.Src)

	encOpts := opts
	encOpts.Width = width
	encOpts.Height = height
	encOpts.ColorType = ColorRGBA
	encOpts.PreserveChunks = preserved
	encOpts.EnsureSizeNotLarger = true
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
