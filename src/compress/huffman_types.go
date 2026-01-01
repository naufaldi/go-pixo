package compress

// Code represents a Huffman code with its bit pattern and length.
// Bits is stored LSB-first (bit-reversed) for DEFLATE compatibility.
type Code struct {
	Bits   uint16
	Length int
}

// Table represents a Huffman code table mapping symbols to codes.
type Table struct {
	Codes     []Code
	MaxLength int
}
