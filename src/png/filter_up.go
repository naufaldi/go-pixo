package png

// ApplyFilterUp applies the "Up" PNG filter to a scanline.
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

// ApplyFilterUpWithScratch writes the filter-type byte and Up-filtered row into the scratch buffer and returns that slice.
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

// ApplyFilterUpWithScratchDst writes the filter-type byte and Up-filtered row into dst, which must hold len(row)+1 bytes.
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
