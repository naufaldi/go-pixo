package png

import (
	"encoding/binary"
	"fmt"
)

var (
	ChunkSRGB ChunkType = "sRGB"
	ChunkICCP ChunkType = "iCCP"
	ChunkGAMA ChunkType = "gAMA"
	ChunkCHRM ChunkType = "cHRM"
)

func ExtractPreserveChunks(pngBytes []byte) ([]Chunk, error) {
	chunks, err := ReadChunks(pngBytes)
	if err != nil {
		return nil, err
	}

	var preserved []Chunk
	for _, c := range chunks {
		switch c.chunkType {
		case ChunkSRGB, ChunkICCP, ChunkGAMA, ChunkCHRM:
			preserved = append(preserved, c)
		}
	}

	return preserved, nil
}

func ReadChunks(pngBytes []byte) ([]Chunk, error) {
	signature := Signature()
	if len(pngBytes) < len(signature) {
		return nil, fmt.Errorf("png: input too small")
	}
	if !bytesEqual(pngBytes[:len(signature)], signature) {
		return nil, fmt.Errorf("png: invalid signature")
	}

	i := len(signature)
	var chunks []Chunk
	for {
		if i+8 > len(pngBytes) {
			return nil, fmt.Errorf("png: truncated chunk header")
		}
		length := int(binary.BigEndian.Uint32(pngBytes[i : i+4]))
		i += 4
		chunkType := ChunkType(string(pngBytes[i : i+4]))
		i += 4
		if length < 0 || i+length+4 > len(pngBytes) {
			return nil, fmt.Errorf("png: invalid chunk length")
		}
		data := pngBytes[i : i+length]
		i += length
		_ = binary.BigEndian.Uint32(pngBytes[i : i+4]) // CRC (ignored for passthrough)
		i += 4

		chunks = append(chunks, Chunk{
			chunkType: chunkType,
			Data:      append([]byte(nil), data...),
		})

		if chunkType == ChunkIEND {
			return chunks, nil
		}
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
