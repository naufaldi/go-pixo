package png

import (
	"encoding/binary"
	"io"

	"github.com/mac/go-pixo/src/compress"
)

type Chunk struct {
	chunkType ChunkType
	Data      []byte
}

func (c *Chunk) Len() int {
	return len(c.Data)
}

func (c *Chunk) Type() string {
	return string(c.chunkType)
}

func (c *Chunk) CRC() uint32 {
	typeBytes := []byte(c.chunkType)
	combined := append(typeBytes, c.Data...)
	return compress.CRC32(combined)
}

func (c *Chunk) Bytes() []byte {
	length := uint32(len(c.Data))
	typeBytes := []byte(c.chunkType)
	crc := c.CRC()

	result := make([]byte, 4+4+len(c.Data)+4)
	binary.BigEndian.PutUint32(result[0:4], length)
	copy(result[4:8], typeBytes)
	copy(result[8:8+len(c.Data)], c.Data)
	binary.BigEndian.PutUint32(result[8+len(c.Data):], crc)

	return result
}

func (c *Chunk) WriteTo(w io.Writer) (int64, error) {
	bytes := c.Bytes()
	n, err := w.Write(bytes)
	return int64(n), err
}

// IsCritical returns true if the chunk is critical for PNG decoding.
// Critical chunks have an uppercase first letter in their type.
func (c *Chunk) IsCritical() bool {
	if len(c.chunkType) == 0 {
		return false
	}
	// Per PNG spec, the 5th bit of the first byte of the chunk type
	// indicates if it's ancillary (1) or critical (0).
	return (c.chunkType[0] & 0x20) == 0
}

// IsRequired returns true if the chunk is absolutely required for a valid PNG.
// These are IHDR, IDAT, and IEND.
func (c *Chunk) IsRequired() bool {
	return c.chunkType == ChunkIHDR || c.chunkType == ChunkIDAT || c.chunkType == ChunkIEND
}
