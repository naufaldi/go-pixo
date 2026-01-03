package png

import (
	"encoding/binary"
	"io"

	"github.com/mac/go-pixo/src/compress"
)

// WritePLTE writes palette as PLTE chunk.
// Palette must have 1-256 colors for valid PNG.
func WritePLTE(w io.Writer, palette Palette) error {
	if palette.NumColors < 1 {
		return ErrInvalidChunkData
	}
	if palette.NumColors > 256 {
		return ErrInvalidChunkData
	}

	data := make([]byte, 3*palette.NumColors)
	for i := 0; i < palette.NumColors; i++ {
		data[i*3+0] = palette.Colors[i].R
		data[i*3+1] = palette.Colors[i].G
		data[i*3+2] = palette.Colors[i].B
	}

	length := uint32(len(data))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}

	if err := binary.Write(w, nil, []byte("PLTE")); err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	crc := compress.CRC32(append([]byte("PLTE"), data...))
	if err := binary.Write(w, binary.BigEndian, crc); err != nil {
		return err
	}

	return nil
}

// PLTEChunkData returns the raw PLTE chunk data without chunk wrapper.
func PLTEChunkData(palette Palette) []byte {
	if palette.NumColors < 1 || palette.NumColors > 256 {
		return nil
	}

	data := make([]byte, 3*palette.NumColors)
	for i := 0; i < palette.NumColors; i++ {
		data[i*3+0] = palette.Colors[i].R
		data[i*3+1] = palette.Colors[i].G
		data[i*3+2] = palette.Colors[i].B
	}

	return data
}

// ValidatePalette checks if a palette is valid for PNG.
func ValidatePalette(palette Palette) error {
	if palette.NumColors < 1 {
		return ErrInvalidChunkData
	}
	if palette.NumColors > 256 {
		return ErrInvalidChunkData
	}
	return nil
}
