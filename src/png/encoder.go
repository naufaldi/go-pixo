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

	ihdr, err := NewIHDRData(width, height, 8, uint8(colorType))
	if err != nil {
		return nil, err
	}

	_ = ihdr

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

	ihdr, err := NewIHDRData(opts.Width, opts.Height, 8, uint8(opts.ColorType))
	if err != nil {
		return nil, err
	}

	_ = ihdr

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
	colorType := opts.ColorType
	bpp := BytesPerPixel(colorType)
	expectedSize := opts.Width * opts.Height * bpp
	if len(pixels) != expectedSize {
		return nil, fmt.Errorf("png: pixel count mismatch: got %d bytes, want %d", len(pixels), expectedSize)
	}

	processedPixels := pixels

	if opts.ReduceColorType {
		if CanReduceToRGB(processedPixels, opts.Width, opts.Height) {
			var err error
			processedPixels, colorType, err = ReduceToRGB(processedPixels, opts.Width, opts.Height)
			if err != nil {
				return nil, err
			}
			bpp = BytesPerPixel(colorType)
		} else if CanReduceToGrayscale(processedPixels, opts.Width, opts.Height, colorType) {
			var err error
			processedPixels, colorType, err = ReduceToGrayscale(processedPixels, opts.Width, opts.Height, colorType)
			if err != nil {
				return nil, err
			}
			bpp = BytesPerPixel(colorType)
		}
	}

	if opts.OptimizeAlpha && colorType == ColorRGBA {
		processedPixels = OptimizeAlpha(processedPixels, colorType)
	}

	var buf bytes.Buffer

	if err := writeSignature(&buf); err != nil {
		return nil, err
	}

	if err := writeIHDR(&buf, opts.Width, opts.Height, colorType); err != nil {
		return nil, err
	}

	if err := WriteIDATWithOptions(&buf, processedPixels, opts.Width, opts.Height, colorType, opts); err != nil {
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
