package png

import "fmt"

// Error is a typed error used by this package for common PNG failures.
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("png: %s", e.Message)
}

var (
	// ErrInvalidSignature indicates the PNG signature bytes were invalid.
	ErrInvalidSignature = &Error{"invalid PNG signature"}
	// ErrUnknownChunkType indicates a chunk type was not recognized.
	ErrUnknownChunkType = &Error{"unknown chunk type"}
	// ErrInvalidDimensions indicates image dimensions were invalid.
	ErrInvalidDimensions = &Error{"invalid image dimensions"}
	// ErrInvalidChunkData indicates a chunk payload was invalid.
	ErrInvalidChunkData = &Error{"invalid chunk data"}
)
