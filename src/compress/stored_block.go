package compress

import (
	"encoding/binary"
	"io"
)

// StoredBlockError represents errors for stored block operations.
type StoredBlockError string

func (e StoredBlockError) Error() string {
	return string(e)
}

const (
	// ErrInvalidBlockSize is returned when the data size exceeds the maximum for a stored block.
	ErrInvalidBlockSize StoredBlockError = "stored block data size exceeds maximum (65535 bytes)"
)

// storedBlockHeader writes the DEFLATE stored block header.
// The header format is:
//   - Bits 0-2: Block type = 000 (stored/uncompressed)
//   - Bit 3: BFINAL (1 if this is the last block, 0 otherwise)
//   - Bits 4-7: Padding (must be 0)
//
// The header is stored in a single byte where:
//
//	header = (block_type & 0x07) | ((bfinal & 0x01) << 3) | (padding << 4)
//	       = 0x00 | ((bfinal & 0x01) << 3) | 0x00
//	       = (bfinal << 3)
func WriteStoredBlockHeader(w io.Writer, final bool) error {
	var buf [1]byte
	if final {
		buf[0] = 0x01 // BFINAL=1, type=000
	} else {
		buf[0] = 0x00 // BFINAL=0, type=000
	}
	_, err := w.Write(buf[:])
	return err
}

// WriteBlockData writes uncompressed data to the stored block.
func WriteBlockData(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	return err
}

// WriteBlockFooter writes the LEN and NLEN fields for a stored block.
// LEN: 2-byte little-endian length of the data
// NLEN: 2-byte one's complement of LEN (used for error detection)
func WriteBlockFooter(w io.Writer, n uint32) error {
	if n > 65535 {
		return ErrInvalidBlockSize
	}

	var buf [4]byte
	binary.LittleEndian.PutUint16(buf[0:2], uint16(n))
	nlen := ^uint16(n)
	binary.LittleEndian.PutUint16(buf[2:4], nlen)

	_, err := w.Write(buf[:])
	return err
}

// WriteStoredBlock writes a complete stored block (header + data + footer).
func WriteStoredBlock(w io.Writer, data []byte, final bool) error {
	if err := WriteStoredBlockHeader(w, final); err != nil {
		return err
	}
	if err := WriteBlockFooter(w, uint32(len(data))); err != nil {
		return err
	}
	return WriteBlockData(w, data)
}

// StoredBlockBytes returns the byte representation of a stored block.
func StoredBlockBytes(data []byte, final bool) ([]byte, error) {
	if len(data) > 65535 {
		return nil, ErrInvalidBlockSize
	}

	result := make([]byte, 1+4+len(data))

	// Header byte: type=000, BFINAL
	if final {
		result[0] = 0x01
	} else {
		result[0] = 0x00
	}

	// Footer: LEN and NLEN
	binary.LittleEndian.PutUint16(result[1:3], uint16(len(data)))
	nlen := ^uint16(len(data))
	binary.LittleEndian.PutUint16(result[3:5], nlen)

	// Data
	copy(result[5:], data)

	return result, nil
}
