package compress

import (
	"encoding/binary"
	"io"
	"math/bits"
)

type ZlibHeaderError string

func (e ZlibHeaderError) Error() string {
	return string(e)
}

const (
	ErrInvalidWindowSize       ZlibHeaderError = "invalid window size for zlib"
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

func ZlibFooterBytes(checksum uint32) [4]byte {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], checksum)
	return buf
}
