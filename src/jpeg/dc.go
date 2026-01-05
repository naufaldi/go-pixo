package jpeg

import (
	"math/bits"
)

// Category returns the bit length needed to represent a DC difference or AC coefficient.
func Category(value int16) uint8 {
	if value == 0 {
		return 0
	}
	absVal := uint16(value)
	if value < 0 {
		absVal = uint16(-value)
	}
	return uint8(bits.Len16(absVal))
}

// EncodeDC calculates the category and diff bits for differential DC encoding.
// It returns the category (0-11 for DC), the diff bits in one's complement format,
// and the number of bits to write.
func EncodeDC(dc, prevDC int16) (category uint8, diffBits uint16, bitLen uint8) {
	diff := dc - prevDC
	if diff == 0 {
		return 0, 0, 0
	}

	cat := Category(diff)
	bits, len := EncodeValue(diff)
	return cat, bits, len
}

// EncodeValue encodes a coefficient value using one's complement for negative values.
// This is the standard JPEG encoding for DC differences and AC coefficients.
func EncodeValue(value int16) (uint16, uint8) {
	if value == 0 {
		return 0, 0
	}

	cat := Category(value)
	var bits uint16
	if value < 0 {
		// Negative values: use one's complement
		// bits = (value - 1) masked to category length
		bits = uint16(value - 1)
	} else {
		bits = uint16(value)
	}

	// Mask to category length
	bits &= (1 << cat) - 1
	return bits, cat
}

// DecodeDC is a helper for testing that performs the inverse of EncodeDC logic.
func DecodeDC(category uint8, diffBits uint16) int16 {
	if category == 0 {
		return 0
	}

	// If leading bit is 1, it's positive. If 0, it's negative.
	// In one's complement representation used by JPEG:
	// category 2:
	// 00 -> -3
	// 01 -> -2
	// 10 -> 2
	// 11 -> 3
	half := uint16(1 << (category - 1))
	if diffBits >= half {
		return int16(diffBits)
	}

	// Negative value calculation
	return int16(diffBits) - int16((1<<category)-1)
}
