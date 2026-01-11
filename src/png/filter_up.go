package png

func ApplyFilterUp(row []byte, prev []byte) []byte {
	result := make([]byte, len(row))
	for i := 0; i < len(row); i++ {
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		result[i] = row[i] - up
	}
	return result
}

func ApplyFilterUpWithScratch(row []byte, prev []byte, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 2
	for i := 0; i < len(row); i++ {
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		filtered[i+1] = row[i] - up
	}
	return filtered[:len(row)+1]
}

func ApplyFilterUpWithScratchDst(row []byte, prev []byte, dst []byte) []byte {
	dst[0] = 2
	for i := 0; i < len(row); i++ {
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		dst[i+1] = row[i] - up
	}
	return dst[:len(row)+1]
}
