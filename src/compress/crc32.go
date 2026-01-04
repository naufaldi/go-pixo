package compress

import (
	"hash"
	"hash/crc32"
)

// CRC32 computes the IEEE CRC-32 checksum for data.
func CRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// NewCRC32 returns a new hash.Hash32 computing the IEEE CRC-32 checksum.
func NewCRC32() hash.Hash32 {
	return crc32.NewIEEE()
}
