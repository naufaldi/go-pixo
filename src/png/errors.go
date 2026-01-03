package png

import "fmt"

type PngError struct {
	Message string
}

func (e *PngError) Error() string {
	return fmt.Sprintf("png: %s", e.Message)
}

var (
	ErrInvalidSignature  = &PngError{"invalid PNG signature"}
	ErrUnknownChunkType  = &PngError{"unknown chunk type"}
	ErrInvalidDimensions = &PngError{"invalid image dimensions"}
	ErrInvalidChunkData  = &PngError{"invalid chunk data"}
)
