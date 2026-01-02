package compress

// LiteralLengthTable returns the RFC1951 fixed Huffman table for literal/length symbols (0-287).
// The table uses predefined code lengths:
//   - Symbols 0-143: 8 bits
//   - Symbols 144-255: 9 bits
//   - Symbols 256-279: 7 bits (length codes)
//   - Symbols 280-287: 8 bits
func LiteralLengthTable() Table {
	lengths := make([]int, 288)

	for i := 0; i < 144; i++ {
		lengths[i] = 8
	}
	for i := 144; i < 256; i++ {
		lengths[i] = 9
	}
	for i := 256; i < 280; i++ {
		lengths[i] = 7
	}
	for i := 280; i < 288; i++ {
		lengths[i] = 8
	}

	codes, _ := buildTableFromLengths(lengths)
	maxLength := 0
	for _, length := range lengths {
		if length > maxLength {
			maxLength = length
		}
	}

	return Table{
		Codes:     codes,
		MaxLength: maxLength,
	}
}

// DistanceTable returns the RFC1951 fixed Huffman table for distance symbols (0-29).
// All distance codes use 5 bits.
func DistanceTable() Table {
	lengths := make([]int, 30)
	for i := 0; i < 30; i++ {
		lengths[i] = 5
	}

	codes, _ := buildTableFromLengths(lengths)

	return Table{
		Codes:     codes,
		MaxLength: 5,
	}
}

// buildTableFromLengths builds canonical Huffman codes from code lengths.
func buildTableFromLengths(lengths []int) ([]Code, []int) {
	codesMap := make(map[int]Code)

	for symbol, length := range lengths {
		if length > 0 {
			codesMap[symbol] = Code{Length: length}
		}
	}

	return Canonicalize(codesMap)
}
