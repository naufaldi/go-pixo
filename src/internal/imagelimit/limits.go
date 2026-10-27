// Package imagelimit bounds image input and raster workspaces before allocation.
package imagelimit

import "fmt"

const (
	// MaxInputBytes matches the browser's 64 MiB compressed-file limit.
	MaxInputBytes = 64 * 1024 * 1024
	// MaxPixels bounds a decoded 8-byte raster to 192 MB, or RGBA to 96 MB.
	MaxPixels = 24_000_000
	// MaxDimension also bounds scanline and codec padding workspaces.
	MaxDimension = 16_384
)

// CheckEncodedSize rejects empty and oversized files before parsing or copying.
func CheckEncodedSize(size int) error {
	if size <= 0 || size > MaxInputBytes {
		return fmt.Errorf("image: encoded input exceeds limit (1 byte to 64 MiB)")
	}
	return nil
}

// CheckDimensions uses division before multiplication, including on 32-bit WASM.
func CheckDimensions(width, height int) error {
	if width <= 0 || height <= 0 || width > MaxDimension || height > MaxDimension || width > MaxPixels/height {
		return fmt.Errorf("image: dimensions exceed limit (24 megapixels, 16384 pixels per side)")
	}
	return nil
}

// PixelBytes returns a checked raster size for supported 1 to 8-byte pixels.
func PixelBytes(width, height, bytesPerPixel int) (int, error) {
	if err := CheckDimensions(width, height); err != nil {
		return 0, err
	}
	if bytesPerPixel < 1 || bytesPerPixel > 8 {
		return 0, fmt.Errorf("image: unsupported bytes per pixel: %d", bytesPerPixel)
	}
	// At most 24M * 8, safely below the 32-bit int limit.
	return width * height * bytesPerPixel, nil
}
