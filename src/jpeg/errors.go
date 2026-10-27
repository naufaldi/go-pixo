package jpeg

import "fmt"

// JpegError represents an error that occurred during JPEG encoding.
type JpegError struct { //nolint:revive // renaming to jpeg.Error would break the exported API
	Message string
}

func (e *JpegError) Error() string {
	return fmt.Sprintf("jpeg: %s", e.Message)
}

// Sentinel errors returned by the JPEG encoder for invalid input.
var (
	ErrInvalidQuality       = &JpegError{"quality must be between 1 and 100"}
	ErrInvalidDimensions    = &JpegError{"invalid image dimensions"}
	ErrUnsupportedColorType = &JpegError{"unsupported color type"}
	ErrInvalidDataLength    = &JpegError{"invalid data length"}
)
