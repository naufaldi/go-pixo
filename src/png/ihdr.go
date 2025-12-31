package png

import (
	"encoding/binary"
	"fmt"
	"io"
)

type IHDRData struct {
	Width       uint32
	Height      uint32
	BitDepth    uint8
	ColorType   ColorType
	Compression uint8
	Filter      uint8
	Interlace   uint8
}

func NewIHDRData(width, height int, bitDepth, colorType uint8) (*IHDRData, error) {
	ihdr := &IHDRData{
		Width:       uint32(width),
		Height:      uint32(height),
		BitDepth:    bitDepth,
		ColorType:   ColorType(colorType),
		Compression: 0,
		Filter:      0,
		Interlace:   0,
	}

	if err := ihdr.Validate(); err != nil {
		return nil, err
	}

	return ihdr, nil
}

func (i *IHDRData) Bytes() []byte {
	result := make([]byte, 13)
	binary.LittleEndian.PutUint32(result[0:4], i.Width)
	binary.LittleEndian.PutUint32(result[4:8], i.Height)
	result[8] = i.BitDepth
	result[9] = uint8(i.ColorType)
	result[10] = i.Compression
	result[11] = i.Filter
	result[12] = i.Interlace
	return result
}

func (i *IHDRData) Validate() error {
	if i.Width == 0 || i.Height == 0 {
		return ErrInvalidDimensions
	}

	if i.Width > 0x7FFFFFFF || i.Height > 0x7FFFFFFF {
		return fmt.Errorf("png: dimensions exceed maximum (2^31-1)")
	}

	validBitDepths := map[ColorType][]uint8{
		ColorGrayscale: {1, 2, 4, 8, 16},
		ColorRGB:       {8, 16},
		ColorRGBA:      {8, 16},
	}

	allowedDepths, ok := validBitDepths[i.ColorType]
	if !ok {
		return fmt.Errorf("png: invalid color type %d", i.ColorType)
	}

	validDepth := false
	for _, depth := range allowedDepths {
		if i.BitDepth == depth {
			validDepth = true
			break
		}
	}

	if !validDepth {
		return fmt.Errorf("png: bit depth %d not valid for color type %d", i.BitDepth, i.ColorType)
	}

	if i.Compression != 0 {
		return fmt.Errorf("png: compression method must be 0 (DEFLATE)")
	}

	if i.Filter != 0 {
		return fmt.Errorf("png: filter method must be 0")
	}

	if i.Interlace > 1 {
		return fmt.Errorf("png: interlace method must be 0 or 1")
	}

	return nil
}

func WriteIHDR(w io.Writer, data *IHDRData) error {
	chunk := &Chunk{
		chunkType: ChunkIHDR,
		Data:      data.Bytes(),
	}

	_, err := chunk.WriteTo(w)
	return err
}
