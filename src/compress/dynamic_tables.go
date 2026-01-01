package compress

// BuildDynamicTables builds canonical Huffman tables from literal/length and distance frequencies.
// Returns the literal/length table and distance table.
func BuildDynamicTables(litFreq []int, distFreq []int) (litTable Table, distTable Table) {
	litTree := BuildTree(litFreq)
	var litCodes []Code
	var litLengths []int
	if litTree != nil {
		codesMap := GenerateCodes(litTree)
		litCodes, litLengths = Canonicalize(codesMap)
	}
	
	maxLitLength := 0
	for _, length := range litLengths {
		if length > maxLitLength {
			maxLitLength = length
		}
	}
	
	distTree := BuildTree(distFreq)
	var distCodes []Code
	var distLengths []int
	if distTree != nil {
		codesMap := GenerateCodes(distTree)
		distCodes, distLengths = Canonicalize(codesMap)
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
