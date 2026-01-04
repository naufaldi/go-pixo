package compress

import (
	"encoding/binary"
	"io"
	"math/bits"
)

// ZlibHeaderError represents an error related to building or writing zlib headers.
type ZlibHeaderError string

// Error implements error.
func (e ZlibHeaderError) Error() string {
	return string(e)
}

const (
	// ErrInvalidWindowSize is returned when the zlib window size is invalid.
	ErrInvalidWindowSize ZlibHeaderError = "invalid window size for zlib"
	// ErrInvalidCompressionLevel is returned when the zlib compression level is invalid.
	ErrInvalidCompressionLevel ZlibHeaderError = "invalid compression level for zlib"
)

func cmfByte(windowSize int) (byte, error) {
	if windowSize&(windowSize-1) != 0 {
		return 0, ErrInvalidWindowSize
	}

	wlog := bits.TrailingZeros(uint(windowSize))
	cinfo := wlog - 8
	if cinfo < 0 || cinfo > 7 {
		return 0, ErrInvalidWindowSize
	}

	cm := 8
	return byte((cm & 0x0F) | ((cinfo & 0x0F) << 4)), nil
}

// WriteCMF writes the zlib CMF byte for a given window size.
func WriteCMF(w io.Writer, windowSize int) error {
	cmf, err := cmfByte(windowSize)
	if err != nil {
		return err
	}

	var buf [1]byte
	buf[0] = cmf
	_, err = w.Write(buf[:])
	return err
}

// WriteFLG writes the zlib FLG byte based on CMF and compression level.
func WriteFLG(w io.Writer, cmf byte, level uint8) error {
	if level > 3 {
		return ErrInvalidCompressionLevel
	}

	fdict := uint8(0)
	flevel := level & 3
	base := (flevel << 6) | ((fdict & 1) << 5)

	fcheck := 31 - ((int(cmf)*256 + int(base)) % 31)
	if fcheck == 31 {
		fcheck = 0
	}

	flg := base | uint8(fcheck)
	var buf [1]byte
	buf[0] = flg
	_, err := w.Write(buf[:])
	return err
}

// WriteZlibHeader writes a complete zlib header (CMF + FLG).
func WriteZlibHeader(w io.Writer, windowSize int, level uint8) error {
	cmf, err := cmfByte(windowSize)
	if err != nil {
		return err
	}
	if err := WriteCMF(w, windowSize); err != nil {
		return err
	}
	return WriteFLG(w, cmf, level)
}

// ZlibHeaderBytes returns the zlib header (CMF + FLG) bytes for a given window size and level.
func ZlibHeaderBytes(windowSize int, level uint8) ([]byte, error) {
	if level > 3 {
		return nil, ErrInvalidCompressionLevel
	}

	var buf [2]byte
	cmf, err := cmfByte(windowSize)
	if err != nil {
		return nil, err
	}
	buf[0] = cmf

	fdict := uint8(0)
	flevel := level & 3
	base := (flevel << 6) | ((fdict & 1) << 5)

	fcheck := 31 - ((int(cmf)*256 + int(base)) % 31)
	if fcheck == 31 {
		fcheck = 0
	}

	buf[1] = base | uint8(fcheck)
	return buf[:], nil
}

// ZlibFooterBytes returns the zlib footer bytes (big-endian Adler-32 checksum).
func ZlibFooterBytes(checksum uint32) [4]byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], checksum)
	return buf
}
