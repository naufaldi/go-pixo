package compress

// Block type constants for DEFLATE blocks.
const (
	BlockTypeStored  = 0 // 00 - uncompressed stored block
	BlockTypeFixed   = 1 // 01 - fixed Huffman codes
	BlockTypeDynamic = 2 // 10 - dynamic Huffman codes
)

// LengthBase contains the base length values for length codes 257-285.
// Length = LengthBase[code-257] + extra bits
var LengthBase = [29]uint16{
	3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31,
	35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258,
}

// LengthExtraBits contains the number of extra bits to read for each length code.
var LengthExtraBits = [29]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2,
	3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0,
}

// DistanceBase contains the base distance values for distance codes 0-29.
// Distance = DistanceBase[code] + extra bits
var DistanceBase = [30]uint16{
	1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193,
	257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145,
	8193, 12289, 16385, 24577,
}

// DistanceExtraBits contains the number of extra bits to read for each distance code.
var DistanceExtraBits = [30]uint8{
	0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6,
	7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13,
}

// EndOfBlockSymbol is the symbol that marks the end of a DEFLATE block.
const EndOfBlockSymbol = 256

// MinMatchLength is the minimum match length for DEFLATE (3 bytes).
const MinMatchLength = 3

// MaxMatchLength is the maximum match length for DEFLATE (258 bytes).
const MaxMatchLength = 258

// MaxDistance is the maximum back-reference distance for DEFLATE (32768 bytes).
const MaxDistance = 32768
