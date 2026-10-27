package png

// ApplyFilterNone returns an unfiltered copy of row.
func ApplyFilterNone(row []byte) []byte {
	result := make([]byte, len(row))
	copy(result, row)
	return result
}

// ApplyFilterNoneWithScratch writes the filter-type byte and unfiltered row into the scratch buffer and returns that slice.
func ApplyFilterNoneWithScratch(row []byte, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 0
	copy(filtered[1:], row)
	return filtered[:len(row)+1]
}

// ApplyFilterNoneWithScratchDst writes the filter-type byte and unfiltered row into dst, which must hold len(row)+1 bytes.
func ApplyFilterNoneWithScratchDst(row []byte, dst []byte) []byte {
	dst[0] = 0
	copy(dst[1:], row)
	return dst[:len(row)+1]
}
