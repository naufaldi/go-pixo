package compress

import "hash"

// Adler32 is the recommended modulus for Adler-32 checksums.
const adler32Mod = 65521

// Adler32 computes the Adler-32 checksum of data.
// This follows RFC 1950 algorithm.
func Adler32(data []byte) uint32 {
	if len(data) == 0 {
		return 1
	}

	s1 := uint32(1)
	s2 := uint32(0)

	for _, b := range data {
		s1 = (s1 + uint32(b)) % adler32Mod
		s2 = (s2 + s1) % adler32Mod
	}

	return s2<<16 | s1
}

// adler32Writer implements hash.Hash32 for streaming Adler32 computation.
type adler32Writer struct {
	s1 uint32
	s2 uint32
}

// NewAdler32 returns a new hash.Hash32 computing the Adler-32 checksum.
func NewAdler32() hash.Hash32 {
	return &adler32Writer{s1: 1, s2: 0}
}

func (a *adler32Writer) Write(p []byte) (n int, err error) {
	for _, b := range p {
		a.s1 = (a.s1 + uint32(b)) % adler32Mod
		a.s2 = (a.s2 + a.s1) % adler32Mod
	}
	return len(p), nil
}

func (a *adler32Writer) Sum(b []byte) []byte {
	checksum := a.s2<<16 | a.s1
	sum := make([]byte, 4)
	sum[0] = byte(checksum >> 24)
	sum[1] = byte(checksum >> 16)
	sum[2] = byte(checksum >> 8)
	sum[3] = byte(checksum)
	return append(b, sum...)
}

func (a *adler32Writer) Reset() {
	a.s1 = 1
	a.s2 = 0
}

func (a *adler32Writer) Size() int     { return 4 }
func (a *adler32Writer) BlockSize() int { return 1 }
func (a *adler32Writer) Sum32() uint32 { return a.s2<<16 | a.s1 }
