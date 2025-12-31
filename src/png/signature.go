package png

import "bytes"

func IsValidSignature(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	return bytes.Equal(data[:8], PNG_SIGNATURE[:])
}

func Signature() []byte {
	return PNG_SIGNATURE[:]
}
