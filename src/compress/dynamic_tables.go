package compress

// BuildDynamicTables builds canonical Huffman tables from literal/length and distance frequencies.
// Returns the literal/length table and distance table.
// The tables are sized to accommodate all possible DEFLATE symbols (0-286 for literal/length, 0-29 for distance).
func BuildDynamicTables(litFreq []int, distFreq []int) (litTable Table, distTable Table) {
	litTree := BuildTree(litFreq)
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
	
	distTree := BuildTree(distFreq)
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
