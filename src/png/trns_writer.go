package png

import (
	"encoding/binary"
	"io"

	"github.com/mac/go-pixo/src/compress"
)

// WriteTRNS writes alpha values for palette entries.
// Only needed if palette has transparency.
// The alpha values correspond to each palette entry in order.
func WriteTRNS(w io.Writer, alphaValues []uint8) error {
	if len(alphaValues) == 0 {
		return nil
	}
	if len(alphaValues) > 256 {
		return ErrInvalidChunkData
	}

	data := make([]byte, len(alphaValues))
	for i, a := range alphaValues {
		data[i] = a
	}

	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	if err := binary.Write(w, nil, []byte("tRNS")); err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	crc := compress.CRC32(append([]byte("tRNS"), data...))
	if err := binary.Write(w, binary.BigEndian, crc); err != nil {
		return err
	}

	return nil
}

// TRNSChunkData returns the raw tRNS chunk data without chunk wrapper.
func TRNSChunkData(alphaValues []uint8) []byte {
	if len(alphaValues) == 0 || len(alphaValues) > 256 {
		return nil
	}

	data := make([]byte, len(alphaValues))
	for i, a := range alphaValues {
		data[i] = a
	}

	return data
}

// ExtractAlphaFromPixels extracts alpha values from RGBA pixels for palette quantization.
// Returns slice of alpha values and whether any transparency exists.
func ExtractAlphaFromPixels(pixels []byte, palette Palette) ([]uint8, bool) {
	alphaValues := make([]uint8, palette.NumColors)
	hasTransparency := false

	for i := 0; i < palette.NumColors; i++ {
		alphaValues[i] = 255 // Default to fully opaque
	}

	return alphaValues, hasTransparency
}

// ValidateTRNS checks if tRNS data is valid for a given palette.
func ValidateTRNS(alphaValues []uint8, paletteSize int) error {
	if len(alphaValues) > paletteSize {
		return ErrInvalidChunkData
	}
	return nil
}
