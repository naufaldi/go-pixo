package compress

import (
	"encoding/binary"
	"io"
)

// WriteAdler32Footer writes the zlib Adler-32 footer checksum in big-endian.
func WriteAdler32Footer(w io.Writer, checksum uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], checksum)
	_, err := w.Write(buf[:])
	return err
}
