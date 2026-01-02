package compress

// BuildDynamicTables builds canonical Huffman tables from literal/length and distance frequencies.
// Returns the literal/length table and distance table.
// The tables are sized to accommodate all possible DEFLATE symbols (0-286 for literal/length, 0-29 for distance).
// Ensures at least 2 symbols in each frequency table to avoid degenerate single-symbol trees.
func BuildDynamicTables(litFreq []int, distFreq []int) (litTable Table, distTable Table) {
	litFreqFixed := ensureAtLeastTwoSymbols(litFreq, 287)
	litTree := BuildTree(litFreqFixed)
	litCodes := make([]Code, 287)
	litLengths := make([]int, 287)
	if litTree != nil {
		codesMap := GenerateCodes(litTree)
		canonCodes, canonLengths := Canonicalize(codesMap)
		if canonCodes != nil {
			copy(litCodes, canonCodes)
			copy(litLengths, canonLengths)
		}
	}

	maxLitLength := 0
	for _, length := range litLengths {
		if length > maxLitLength {
			maxLitLength = length
		}
	}

	distFreqFixed := ensureAtLeastTwoSymbols(distFreq, 30)
	distTree := BuildTree(distFreqFixed)
	distCodes := make([]Code, 30)
	distLengths := make([]int, 30)
	if distTree != nil {
		codesMap := GenerateCodes(distTree)
		canonCodes, canonLengths := Canonicalize(codesMap)
		if canonCodes != nil {
			copy(distCodes, canonCodes)
			copy(distLengths, canonLengths)
		}
	}

	maxDistLength := 0
	for _, length := range distLengths {
		if length > maxDistLength {
			maxDistLength = length
		}
	}

	litTable = Table{
		Codes:     litCodes,
		MaxLength: maxLitLength,
	}

	distTable = Table{
		Codes:     distCodes,
		MaxLength: maxDistLength,
	}

	return litTable, distTable
}

// ensureAtLeastTwoSymbols ensures the frequency table has at least 2 non-zero entries.
// If only one symbol has non-zero frequency, injects a dummy second symbol (first unused symbol).
// This prevents degenerate single-symbol Huffman trees that would produce zero-length codes.
func ensureAtLeastTwoSymbols(freq []int, maxSymbol int) []int {
	result := make([]int, maxSymbol)
	copy(result, freq)
	if len(result) > maxSymbol {
		result = result[:maxSymbol]
	}

	nonZeroCount := 0
	firstNonZero := -1
	for i, f := range result {
		if f > 0 {
			nonZeroCount++
			if firstNonZero == -1 {
				firstNonZero = i
			}
		}
	}

	if nonZeroCount == 0 {
		result[0] = 1
		result[1] = 1
		return result
	}

	if nonZeroCount == 1 {
		dummySymbol := firstNonZero + 1
		if dummySymbol >= len(result) {
			dummySymbol = 0
			if dummySymbol == firstNonZero {
				dummySymbol = 1
			}
		}
		if dummySymbol == firstNonZero {
			for i := range result {
				if i != firstNonZero && result[i] == 0 {
					dummySymbol = i
					break
				}
			}
		}
		if dummySymbol < len(result) && dummySymbol != firstNonZero {
			result[dummySymbol] = 1
		}
	}

	return result
}
