package compress

import "io"

// WriteStoredBlockDeflate writes a stored (uncompressed) DEFLATE block.
// This wraps the existing stored block implementation with the expected signature (final, data).
func WriteStoredBlockDeflate(w io.Writer, final bool, data []byte) error {
	return WriteStoredBlock(w, data, final)
}

// WriteFixedBlock writes a fixed Huffman DEFLATE block.
// Tokens are encoded using the RFC1951 fixed Huffman tables.
func WriteFixedBlock(w io.Writer, final bool, tokens []Token) error {
	bw := NewBitWriter(w)

	var blockHeader uint16
	if final {
		blockHeader |= 0x01
	}
	blockHeader |= BlockTypeFixed << 1
	if err := bw.Write(blockHeader, 3); err != nil {
		return err
	}

	litTable := LiteralLengthTable()
	distTable := DistanceTable()

	for _, token := range tokens {
		if token.IsLiteral {
			if err := EncodeLiteral(bw, int(token.Literal), litTable); err != nil {
				return err
			}
		} else {
			length := int(token.Match.Length)
			distance := int(token.Match.Distance)

			if err := EncodeLength(bw, length, litTable); err != nil {
				return err
			}
			if err := EncodeDistance(bw, distance, distTable); err != nil {
				return err
			}
		}
	}

	if err := EncodeLiteral(bw, EndOfBlockSymbol, litTable); err != nil {
		return err
	}

	return bw.Flush()
}

// WriteDynamicBlock writes a dynamic Huffman DEFLATE block.
// Tokens are encoded using custom Huffman tables built from token frequencies.
func WriteDynamicBlock(w io.Writer, final bool, tokens []Token) error {
	bw := NewBitWriter(w)

	var blockHeader uint16
	if final {
		blockHeader |= 0x01
	}
	blockHeader |= BlockTypeDynamic << 1
	if err := bw.Write(blockHeader, 3); err != nil {
		return err
	}

	litFreq, distFreq := countTokenFrequencies(tokens)
	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	litLengths := extractCodeLengths(litTable)
	distLengths := extractCodeLengths(distTable)

	if err := WriteDynamicHeader(bw, litLengths, distLengths); err != nil {
		return err
	}

	for _, token := range tokens {
		if token.IsLiteral {
			if err := EncodeLiteral(bw, int(token.Literal), litTable); err != nil {
				return err
			}
		} else {
			length := int(token.Match.Length)
			distance := int(token.Match.Distance)

			if err := EncodeLength(bw, length, litTable); err != nil {
				return err
			}
			if err := EncodeDistance(bw, distance, distTable); err != nil {
				return err
			}
		}
	}

	if err := EncodeLiteral(bw, EndOfBlockSymbol, litTable); err != nil {
		return err
	}

	return bw.Flush()
}

// countTokenFrequencies counts frequencies of literal/length and distance symbols from tokens.
func countTokenFrequencies(tokens []Token) ([]int, []int) {
	litFreq := make([]int, 287)
	distFreq := make([]int, 30)

	for _, token := range tokens {
		if token.IsLiteral {
			litFreq[int(token.Literal)]++
		} else {
			length := int(token.Match.Length)
			lengthCode := findLengthCode(length)
			if lengthCode >= 257 && lengthCode <= 285 {
				litFreq[lengthCode]++
			}

			distance := int(token.Match.Distance)
			distanceCode := findDistanceCode(distance)
			if distanceCode >= 0 && distanceCode < 30 {
				distFreq[distanceCode]++
			}
		}
	}

	litFreq[EndOfBlockSymbol] = 1

	return litFreq, distFreq
}

// extractCodeLengths extracts code lengths from a Huffman table.
func extractCodeLengths(table Table) []int {
	lengths := make([]int, len(table.Codes))
	for i, code := range table.Codes {
		if i < len(lengths) {
			lengths[i] = code.Length
		}
	}
	return lengths
}
