package jpeg

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// jpegOrientation reads only IFD0's inline SHORT orientation before scan data.
// It never follows thumbnail, maker-note, or next-IFD pointers.
func jpegOrientation(input []byte) (uint16, error) {
	invalid := func() (uint16, error) { return 0, errors.New("jpeg: invalid EXIF orientation metadata") }
	if len(input) < 2 || input[0] != 0xff || input[1] != 0xd8 {
		return invalid()
	}
	orientation := uint16(1)
	exifSeen := false
	for offset := 2; offset < len(input); {
		if input[offset] != 0xff {
			return invalid()
		}
		for offset < len(input) && input[offset] == 0xff {
			offset++
		}
		if offset == len(input) {
			return invalid()
		}
		marker := input[offset]
		offset++
		if marker == 0xda || marker == 0xd9 {
			return orientation, nil
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if len(input)-offset < 2 {
			return invalid()
		}
		length := int(binary.BigEndian.Uint16(input[offset:]))
		if length < 2 || length > len(input)-offset {
			return invalid()
		}
		segment := input[offset+2 : offset+length]
		offset += length
		if marker != 0xe1 || !bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			continue
		}
		if exifSeen {
			return invalid()
		}
		exifSeen = true
		tiff := segment[6:]
		if len(tiff) < 8 {
			return invalid()
		}
		var order binary.ByteOrder
		switch string(tiff[:2]) {
		case "II":
			order = binary.LittleEndian
		case "MM":
			order = binary.BigEndian
		default:
			return invalid()
		}
		if order.Uint16(tiff[2:]) != 42 {
			return invalid()
		}
		// Compare before converting to int: WASM uses 32-bit ints.
		ifd := order.Uint32(tiff[4:])
		if ifd < 8 || uint64(ifd)+2 > uint64(len(tiff)) {
			return invalid()
		}
		count := int(order.Uint16(tiff[int(ifd):]))
		if count > 1024 || uint64(ifd)+2+uint64(count)*12+4 > uint64(len(tiff)) {
			return invalid()
		}
		found := false
		for index := 0; index < count; index++ {
			entry := tiff[int(ifd)+2+index*12:]
			if order.Uint16(entry) != 0x0112 {
				continue
			}
			if found || order.Uint16(entry[2:]) != 3 || order.Uint32(entry[4:]) != 1 {
				return invalid()
			}
			orientation = order.Uint16(entry[8:])
			if orientation < 1 || orientation > 8 {
				return invalid()
			}
			found = true
		}
	}
	return invalid()
}

func orientationSource(x, y, width, height int, orientation uint16) (int, int) {
	switch orientation {
	case 2:
		return width - 1 - x, y
	case 3:
		return width - 1 - x, height - 1 - y
	case 4:
		return x, height - 1 - y
	case 5:
		return y, x
	case 6:
		return y, height - 1 - x
	case 7:
		return width - 1 - y, height - 1 - x
	case 8:
		return width - 1 - y, x
	default:
		return x, y
	}
}
