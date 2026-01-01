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

	return &Encoder{
		width:     width,
		height:    height,
		colorType: colorType,
	}, nil
}

func (e *Encoder) Encode(pixels []byte) ([]byte, error) {
	bpp := BytesPerPixel(e.colorType)
	expectedSize := e.width * e.height * bpp
	if len(pixels) != expectedSize {
		return nil, fmt.Errorf("png: pixel count mismatch: got %d bytes, want %d", len(pixels), expectedSize)
	}

	var buf bytes.Buffer

	if err := e.writeSignature(&buf); err != nil {
		return nil, err
	}

	if err := e.writeIHDR(&buf); err != nil {
		return nil, err
	}

	if err := e.writeIDAT(&buf, pixels); err != nil {
		return nil, err
	}

	if err := e.writeIEND(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *Encoder) writeSignature(w io.Writer) error {
	_, err := w.Write(Signature())
	return err
}

func (e *Encoder) writeIHDR(w io.Writer) error {
	ihdr, err := NewIHDRData(e.width, e.height, 8, uint8(e.colorType))
	if err != nil {
		return err
	}

	return WriteIHDR(w, ihdr)
}

func (e *Encoder) writeIDAT(w io.Writer, pixels []byte) error {
	return WriteIDAT(w, pixels, e.width, e.height, e.colorType)
}

func (e *Encoder) writeIEND(w io.Writer) error {
	return WriteIEND(w)
}

func bytesPerPixel(colorType ColorType) int {
	switch colorType {
	case ColorRGB:
		return 3
	case ColorRGBA:
		return 4
	case ColorGrayscale:
		return 1
	default:
		return 0
	}
}
