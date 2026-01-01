package png

import "io"

// WriteIEND writes the IEND chunk to the writer.
// IEND marks the end of the PNG data stream and has no data.
// Format: length(4 bytes) + "IEND"(4 bytes) + CRC32(4 bytes)
func WriteIEND(w io.Writer) error {
	chunk := Chunk{chunkType: ChunkIEND, Data: nil}
	_, err := chunk.WriteTo(w)
	return err
}
