package compress

import (
	"encoding/binary"
	"io"
)

func WriteAdler32Footer(w io.Writer, checksum uint32) error {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], checksum)
	_, err := w.Write(buf[:])
	return err
}
