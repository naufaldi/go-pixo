package compress

import "sort"

// GenerateCodes traverses the Huffman tree and generates code lengths for each symbol.
// Returns a map from symbol to Code (with Length set, Bits will be set by Canonicalize).
func GenerateCodes(node *Node) map[int]Code {
	codes := make(map[int]Code)
	if node == nil {
		return codes
	}

	var traverse func(n *Node, depth int)
	traverse = func(n *Node, depth int) {
		if n == nil {
			return
		}
		if n.Left == nil && n.Right == nil {
			codes[n.Symbol] = Code{Length: depth}
			return
		}
		if n.Left != nil {
			traverse(n.Left, depth+1)
		}
		if n.Right != nil {
			traverse(n.Right, depth+1)
		}
	}

	traverse(node, 0)
	return codes
}

// Canonicalize converts code lengths to canonical Huffman codes (RFC 1951).
// Codes are assigned in order: first by length, then by symbol value.
// Bits are stored LSB-first (bit-reversed) for DEFLATE compatibility.
// Returns a dense slice of Codes indexed by symbol, and a slice of code lengths indexed by symbol.
func Canonicalize(codes map[int]Code) ([]Code, []int) {
	if len(codes) == 0 {
		return nil, nil
	}

	type symbolLength struct {
		symbol int
		length int
	}

	var symbols []symbolLength
	maxSymbol := 0
	for symbol, code := range codes {
		if code.Length > 0 {
			symbols = append(symbols, symbolLength{symbol: symbol, length: code.Length})
			if symbol > maxSymbol {
				maxSymbol = symbol
			}
		}
	}

	if len(symbols) == 0 {
		return nil, nil
	}

	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].length != symbols[j].length {
			return symbols[i].length < symbols[j].length
		}
		return symbols[i].symbol < symbols[j].symbol
	})

	maxLength := 0
	for _, sl := range symbols {
		if sl.length > maxLength {
			maxLength = sl.length
		}
	}

	lengthCounts := make([]int, maxLength+1)
	for _, sl := range symbols {
		lengthCounts[sl.length]++
	}

	blCount := make([]int, maxLength+1)
	for _, sl := range symbols {
		blCount[sl.length]++
	}

	nextCode := make([]int, maxLength+1)
	code := 0
	blCount[0] = 0
	for bits := 1; bits <= maxLength; bits++ {
		code = (code + blCount[bits-1]) << 1
		nextCode[bits] = code
	}

	resultCodes := make([]Code, maxSymbol+1)
	resultLengths := make([]int, maxSymbol+1)

	for _, sl := range symbols {
		canonicalValue := nextCode[sl.length]
		nextCode[sl.length]++

		canonicalValue &= (1 << uint(sl.length)) - 1
		lsbFirstBits := ReverseBits(uint16(canonicalValue), sl.length)

		resultCodes[sl.symbol] = Code{
			Bits:   lsbFirstBits,
			Length: sl.length,
		}
		resultLengths[sl.symbol] = sl.length
	}

	return resultCodes, resultLengths
}

// ReverseBits reverses the lower n bits of a value for LSB-first storage.
// For example, if value=0b101 (5) and n=3, returns 0b101 (5) because
// reading LSB-first: bit0=1, bit1=0, bit2=1 -> MSB-first: 101.
// If value=0b010 (2) and n=3, returns 0b010 (2) -> reading LSB-first: 010 -> MSB-first: 010.
func ReverseBits(value uint16, n int) uint16 {
	if n == 0 {
		return 0
	}
	result := uint16(0)
	for i := 0; i < n; i++ {
		if value&(1<<uint(i)) != 0 {
			result |= 1 << uint(n-1-i)
		}
	}
	return result
}
