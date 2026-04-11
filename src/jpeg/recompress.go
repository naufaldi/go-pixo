package jpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	stdjpeg "image/jpeg"
)

// RecompressJPEGBytes recompresses an existing JPEG with the provided options.
// Returns the smaller of re-encoded output and original input.
//
// opts acts as a template; Width, Height, and ColorType are overridden to match
// the decoded image.
func RecompressJPEGBytes(inputJPEG []byte, opts Options) ([]byte, error) {
	img, err := stdjpeg.Decode(bytes.NewReader(inputJPEG))
	if err != nil {
		return nil, fmt.Errorf("jpeg: decode input: %w", err)
	}

	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return nil, ErrInvalidDimensions
	}

	// Convert to raw RGB pixels by drawing into an NRGBA canvas and stripping alpha.
	nrgba := image.NewNRGBA(b)
	draw.Draw(nrgba, b, img, b.Min, draw.Src)

	pixels := make([]byte, w*h*3)
	for i := 0; i < w*h; i++ {
		pixels[i*3] = nrgba.Pix[i*4]
		pixels[i*3+1] = nrgba.Pix[i*4+1]
		pixels[i*3+2] = nrgba.Pix[i*4+2]
	}

	// Override dimensions and color type; caller's other options (quality,
	// subsampling, progressive, etc.) are preserved.
	opts.Width = w
	opts.Height = h
	opts.ColorType = ColorRGB

	encoder, err := NewEncoder(opts)
	if err != nil {
		return nil, fmt.Errorf("jpeg: create encoder: %w", err)
	}

	out, err := encoder.Encode(pixels)
	if err != nil {
		return nil, fmt.Errorf("jpeg: encode: %w", err)
	}

	// Validate output is a decodable JPEG.
	if _, err := stdjpeg.Decode(bytes.NewReader(out)); err != nil {
		return nil, fmt.Errorf("jpeg: output decode failed: %w", err)
	}

	if len(out) >= len(inputJPEG) {
		return inputJPEG, nil
	}
	return out, nil
}
