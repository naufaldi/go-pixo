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

// ErrInvalidJPEG matches every error OptimizeLossless returns for malformed or
// truncated JPEG input.
var ErrInvalidJPEG = &JpegError{"invalid JPEG input"}

// ErrUnsupportedJPEG matches every error OptimizeLossless returns for a valid
// JPEG that uses a feature the lossless optimizer does not handle. The more
// specific sentinels below also match it with errors.Is.
var ErrUnsupportedJPEG = &JpegError{"unsupported JPEG input"}

// Specific unsupported-input sentinels returned by OptimizeLossless.
var (
	ErrUnsupportedJPEGComponents error = unsupportedJPEGError("only 1 or 3 components are supported (CMYK/YCCK is not)")
	ErrUnsupportedJPEGPrecision  error = unsupportedJPEGError("only 8-bit sample precision is supported")
	ErrUnsupportedJPEGArithmetic error = unsupportedJPEGError("arithmetic coding is not supported")
	ErrUnsupportedJPEGProcess    error = unsupportedJPEGError("lossless, hierarchical, DNL or changed-table JPEG processes are not supported")
)

type unsupportedJPEGError string

func (e unsupportedJPEGError) Error() string { return "jpeg: unsupported JPEG input: " + string(e) }

// Is lets errors.Is match each specific sentinel against ErrUnsupportedJPEG.
func (e unsupportedJPEGError) Is(target error) bool { return target == ErrUnsupportedJPEG }
