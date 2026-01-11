package png

func ApplyFilterNone(row []byte) []byte {
	result := make([]byte, len(row))
	copy(result, row)
	return result
}

func ApplyFilterNoneWithScratch(row []byte, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 0
	copy(filtered[1:], row)
	return filtered[:len(row)+1]
}

func ApplyFilterNoneWithScratchDst(row []byte, dst []byte) []byte {
	dst[0] = 0
	copy(dst[1:], row)
	return dst[:len(row)+1]
}
