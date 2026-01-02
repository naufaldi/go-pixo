package compress

// CodeLengthOrder is the order in which code length codes are stored in DEFLATE dynamic headers.
var CodeLengthOrder = [19]int{
	16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
}

const (
	maxDynamicLitLenIndex = 285 // HLIT codes cover literal/length symbols 0..285 (286 codes)
	maxDynamicDistIndex   = 29  // HDIST codes cover distance symbols 0..29 (30 codes)
	minCodeLengthOrderIdx = 3   // HCLEN minimum is 4 codes, so last index >= 3
	maxCodeLengthOrderIdx = 18  // 19 code length codes total (0..18)
	maxCodeLengthCodeLen  = 7   // code length code lengths are stored in 3 bits (0..7)
)

// WriteHLIT writes the HLIT field (5 bits): number of literal/length codes - 257.
func WriteHLIT(w *BitWriter, n int) error {
	if n < 257 || n > 286 {
		return ErrInvalidHLIT
	}
	value := uint16(n - 257)
	return w.Write(value, 5)
}

// WriteHDIST writes the HDIST field (5 bits): number of distance codes - 1.
func WriteHDIST(w *BitWriter, n int) error {
	if n < 1 || n > 30 {
		return ErrInvalidHDIST
	}
	value := uint16(n - 1)
	return w.Write(value, 5)
}

// WriteHCLEN writes the HCLEN field (4 bits): number of code length codes - 4.
func WriteHCLEN(w *BitWriter, n int) error {
	if n < 4 || n > 19 {
		return ErrInvalidHCLEN
	}
	value := uint16(n - 4)
	return w.Write(value, 4)
}

// WriteDynamicHeader writes a complete dynamic Huffman block header to the bit writer.
// It includes HLIT, HDIST, HCLEN, code length code lengths, and RLE-encoded literal/length/distance code lengths.
func WriteDynamicHeader(w *BitWriter, litLengths []int, distLengths []int) error {
	litMax := minInt(maxDynamicLitLenIndex, len(litLengths)-1)
	distMax := minInt(maxDynamicDistIndex, len(distLengths)-1)

	litCount := findLastNonZero(litLengths, litMax)
	// HLIT must include at least symbols 0..256 (end-of-block).
	if litCount < EndOfBlockSymbol {
		litCount = EndOfBlockSymbol
	}
	litCount = minInt(litCount, litMax)

	distCount := findLastNonZero(distLengths, distMax)
	distCount = minInt(distCount, distMax)

	if err := WriteHLIT(w, litCount+1); err != nil {
		return err
	}
	if err := WriteHDIST(w, distCount+1); err != nil {
		return err
	}

	codeLengthLengths := buildCodeLengthLengths(litLengths[:litCount+1], distLengths[:distCount+1])
	clenCount := findLastNonZeroCodeLength(codeLengthLengths)
	// HCLEN must encode at least 4 code length code lengths.
	if clenCount < minCodeLengthOrderIdx {
		clenCount = minCodeLengthOrderIdx
	}
	clenCount = minInt(clenCount, maxCodeLengthOrderIdx)

	if err := WriteHCLEN(w, clenCount+1); err != nil {
		return err
	}

	for i := 0; i <= clenCount; i++ {
		code := CodeLengthOrder[i]
		length := codeLengthLengths[code]
		if length < 0 || length > maxCodeLengthCodeLen {
			return DeflateError("invalid code length code length")
		}
		if err := w.Write(uint16(length), 3); err != nil {
			return err
		}
	}

	codeLengthTable := buildCodeLengthTable(codeLengthLengths)

	if err := writeRLECodeLengths(w, litLengths[:litCount+1], codeLengthTable); err != nil {
		return err
	}

	if err := writeRLECodeLengths(w, distLengths[:distCount+1], codeLengthTable); err != nil {
		return err
	}

	return nil
}

// findLastNonZero finds the last non-zero index in lengths, up to max.
func findLastNonZero(lengths []int, max int) int {
	last := 0
	for i := 0; i < len(lengths) && i <= max; i++ {
		if lengths[i] > 0 {
			last = i
		}
	}
	return last
}

// findLastNonZeroCodeLength finds the last non-zero code length index in CodeLengthOrder.
func findLastNonZeroCodeLength(codeLengthLengths []int) int {
	last := 0
	for i := 0; i < len(CodeLengthOrder); i++ {
		code := CodeLengthOrder[i]
		if codeLengthLengths[code] > 0 {
			last = i
		}
	}
	return last
}

// buildCodeLengthLengths builds the code length code lengths from literal/length and distance code lengths.
// RLE symbols 16, 17, 18 are included to ensure they have codes available for RLE encoding.
func buildCodeLengthLengths(litLengths []int, distLengths []int) []int {
	allLengths := make([]int, 0, len(litLengths)+len(distLengths))
	allLengths = append(allLengths, litLengths...)
	allLengths = append(allLengths, distLengths...)
	freq := make([]int, 19)

	for _, length := range allLengths {
		if length < 16 {
			freq[length]++
		} else if length == 16 {
			freq[16]++
		} else if length == 17 {
			freq[17]++
		} else if length == 18 {
			freq[18]++
		}
	}

	freq[16] = 1
	freq[17] = 1
	freq[18] = 1

	freqFixed := ensureAtLeastTwoSymbols(freq, 19)
	tree := BuildTree(freqFixed)
	if tree == nil {
		return make([]int, 19)
	}

	codesMap := GenerateCodes(tree)
	codes, lengths := Canonicalize(codesMap)

	if codes == nil || lengths == nil {
		return make([]int, 19)
	}

	result := make([]int, 19)
	for i := 0; i < 19 && i < len(lengths); i++ {
		if lengths[i] > 0 {
			// Code length code lengths are stored in 3 bits in the DEFLATE header.
			if lengths[i] > maxCodeLengthCodeLen {
				return make([]int, 19)
			}
			result[i] = lengths[i]
		}
	}

	return result
}

// buildCodeLengthTable builds a Huffman table for code length codes.
// The table must have codes for all symbols 0-18 that appear in codeLengthLengths.
// RLE symbols 16, 17, 18 are included if they appear in codeLengthLengths or if they might be used.
func buildCodeLengthTable(codeLengthLengths []int) Table {
	codes, _ := buildTableFromLengths(codeLengthLengths)

	resultCodes := make([]Code, 19)
	if codes != nil {
		copy(resultCodes, codes)
	}

	maxLength := 0
	for _, length := range codeLengthLengths {
		if length > maxLength {
			maxLength = length
		}
	}

	return Table{Codes: resultCodes, MaxLength: maxLength}
}

// writeRLECodeLengths writes code lengths using RLE encoding (symbols 16, 17, 18).
func writeRLECodeLengths(w *BitWriter, lengths []int, codeLengthTable Table) error {
	for i := 0; i < len(lengths); {
		cur := lengths[i]

		// Zero run
		if cur == 0 {
			run := 0
			for i+run < len(lengths) && lengths[i+run] == 0 {
				run++
			}

			for run > 0 {
				switch {
				case run >= 11:
					// 18: repeat zero 11-138 times (7 extra bits)
					n := run
					if n > 138 {
						n = 138
					}
					if err := EncodeLiteral(w, 18, codeLengthTable); err != nil {
						return err
					}
					if err := w.Write(uint16(n-11), 7); err != nil {
						return err
					}
					run -= n
				case run >= 3:
					// 17: repeat zero 3-10 times (3 extra bits)
					n := run
					if n > 10 {
						n = 10
					}
					if err := EncodeLiteral(w, 17, codeLengthTable); err != nil {
						return err
					}
					if err := w.Write(uint16(n-3), 3); err != nil {
						return err
					}
					run -= n
				default:
					// 0: literal code length 0 (no extra bits)
					if err := EncodeLiteral(w, 0, codeLengthTable); err != nil {
						return err
					}
					run--
				}
			}

			i += 0 // i is advanced below
			// Advance over the zero run we processed.
			for i < len(lengths) && lengths[i] == 0 {
				i++
			}
			continue
		}

		// Non-zero run
		run := 0
		for i+run < len(lengths) && lengths[i+run] == cur {
			run++
		}

		// Emit first occurrence as literal length (0-15).
		if err := EncodeLiteral(w, cur, codeLengthTable); err != nil {
			return err
		}
		run--

		// Remaining occurrences: use 16 for chunks of 3-6, otherwise literal.
		for run > 0 {
			if run >= 3 {
				n := run
				if n > 6 {
					n = 6
				}
				if err := EncodeLiteral(w, 16, codeLengthTable); err != nil {
					return err
				}
				if err := w.Write(uint16(n-3), 2); err != nil {
					return err
				}
				run -= n
				continue
			}
			if err := EncodeLiteral(w, cur, codeLengthTable); err != nil {
				return err
			}
			run--
		}

		i += 0 // i advanced below
		// Advance over the non-zero run we encoded.
		for i < len(lengths) && lengths[i] == cur {
			i++
		}
	}

	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
