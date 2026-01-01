package png

import (
	"fmt"
	"io"
)

// ScanlineError represents errors for scanline operations.
type ScanlineError string

func (e ScanlineError) Error() string {
	return string(e)
}

const (
	// ErrEmptyScanline is returned when an empty scanline is detected.
	ErrEmptyScanline ScanlineError = "scanline cannot be empty"
)

// WriteScanline writes a single scanline with its filter byte.
// A scanline consists of:
//   - 1 byte: Filter type (0-4)
//   - N bytes: Pixel data for the row
//
// The filter byte helps compression by indicating how to interpret
// the relationship between this row and the previous row.
func WriteScanline(w io.Writer, filter FilterType, pixels []byte) error {
	if len(pixels) == 0 {
		return ErrEmptyScanline
	}

	// Write filter byte
	var buf [1]byte
	buf[0] = byte(filter)
	if _, err := w.Write(buf[:]); err != nil {
		return fmt.Errorf("png: failed to write filter byte: %w", err)
	}

	// Write pixel data
	if _, err := w.Write(pixels); err != nil {
		return fmt.Errorf("png: failed to write pixel data: %w", err)
	}

	return nil
}

// ScanlineBytes returns the byte representation of a scanline.
func ScanlineBytes(filter FilterType, pixels []byte) ([]byte, error) {
	if len(pixels) == 0 {
		return nil, ErrEmptyScanline
	}

	result := make([]byte, 1+len(pixels))
	result[0] = byte(filter)
	copy(result[1:], pixels)
	return result, nil
}

// BytesPerPixel returns the number of bytes per pixel for a given color type.
func BytesPerPixel(colorType ColorType) int {
	switch colorType {
	case ColorGrayscale:
		return 1
	case ColorRGB:
		return 3
	case ColorRGBA:
		return 4
	default:
		return 1
	}
}

// ScanlineLength returns the expected length of a scanline for a given width and color type.
func ScanlineLength(width int, colorType ColorType) int {
	bpp := BytesPerPixel(colorType)
	// Each scanline has 1 filter byte + width * bytes per pixel
	return 1 + width*bpp
}

// ValidateScanlineData checks if the pixel data length matches the expected scanline length.
func ValidateScanlineData(pixels []byte, width int, colorType ColorType) error {
	expectedLen := ScanlineLength(width, colorType)
	if len(pixels) != expectedLen {
		return fmt.Errorf("png: scanline data length %d does not match expected %d for width=%d, colorType=%d",
			len(pixels), expectedLen, width, colorType)
	}
	return nil
}
