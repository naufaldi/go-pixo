package compress

import (
	"hash"
	"hash/crc32"
)

func CRC32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func NewCRC32() hash.Hash32 {
	return crc32.NewIEEE()
}
