package compress

import (
	"encoding/binary"
	"io"
)

func WriteCMF(w io.Writer, windowSize int) error {
	cm := 8

	var wlog int
	switch windowSize {
	case 1:
		wlog = 0
	case 2:
		wlog = 1
	case 4:
		wlog = 2
	case 8:
		wlog = 3
	case 16:
		wlog = 4
	case 32:
		wlog = 5
	case 64:
		wlog = 6
	case 128:
		wlog = 7
	case 256:
		wlog = 8
	case 512:
		wlog = 9
	case 1024:
		wlog = 10
	case 2048:
		wlog = 11
	case 4096:
		wlog = 12
	case 8192:
		wlog = 13
	case 16384:
		wlog = 14
	case 32768:
		wlog = 15
	default:
		return ErrInvalidWindowSize
	}

	cmf := byte((cm & 0xF) | ((wlog & 0xF) << 4))
	return binary.Write(w, binary.BigEndian, cmf)
}

func WriteFLG(w io.Writer, checksum uint8) error {
	dictFlag := uint8(0)
	level := uint8(2)

	flg := byte((checksum & 0x1F) | ((dictFlag & 1) << 5) | ((level & 3) << 6))
	return binary.Write(w, binary.BigEndian, flg)
}

type ZlibHeaderError string

func (e ZlibHeaderError) Error() string {
	return string(e)
}

const ErrInvalidWindowSize ZlibHeaderError = "invalid window size for zlib"
