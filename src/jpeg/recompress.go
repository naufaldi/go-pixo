package jpeg

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	stdjpeg "image/jpeg"

	"github.com/mac/go-pixo/src/internal/imagelimit"
)

// RecompressJPEGBytes recompresses an existing JPEG with the provided options.
// Returns the smaller of re-encoded output and original input.
//
// opts acts as a template; Width, Height, and ColorType are overridden to match
// the display-oriented image. Re-encoded output applies EXIF orientation to its
// pixels and strips metadata; an original-byte fallback retains its metadata.
func RecompressJPEGBytes(inputJPEG []byte, opts Options) ([]byte, error) {
	if err := imagelimit.CheckEncodedSize(len(inputJPEG)); err != nil {
		return nil, err
	}
	config, err := stdjpeg.DecodeConfig(bytes.NewReader(inputJPEG))
	if err != nil {
		return nil, fmt.Errorf("jpeg: decode config: %w", err)
	}
	if err = imagelimit.CheckDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}
	orientation, err := jpegOrientation(inputJPEG)
	if err != nil {
		return nil, err
	}

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
	if orientation >= 5 {
		w, h = h, w
	}

	pixels := make([]byte, w*h*3)
	if orientation == 1 {
		for i := 0; i < w*h; i++ {
			pixels[i*3] = nrgba.Pix[i*4]
			pixels[i*3+1] = nrgba.Pix[i*4+1]
			pixels[i*3+2] = nrgba.Pix[i*4+2]
		}
	} else {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				sx, sy := orientationSource(x, y, b.Dx(), b.Dy(), orientation)
				source := sy*nrgba.Stride + sx*4
				target := (y*w + x) * 3
				copy(pixels[target:target+3], nrgba.Pix[source:source+3])
			}
		}
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
