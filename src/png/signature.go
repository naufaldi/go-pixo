package png

import "bytes"

// IsValidSignature reports whether data starts with the PNG signature bytes.
func IsValidSignature(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return bytes.Equal(data[:8], pngSignature[:])
}

// Signature returns the PNG signature bytes.
func Signature() []byte {
	return pngSignature[:]
}
