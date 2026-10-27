package png

// ApplyFilterSub applies the "Sub" PNG filter to a scanline.
func ApplyFilterSub(row []byte, bpp int) []byte {
	result := make([]byte, len(row))
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		result[i] = row[i] - left
	}
	return result
}

// ApplyFilterSubWithScratch writes the filter-type byte and Sub-filtered row into the scratch buffer and returns that slice.
func ApplyFilterSubWithScratch(row []byte, bpp int, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 1
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		filtered[i+1] = row[i] - left
	}
	return filtered[:len(row)+1]
}

// ApplyFilterSubWithScratchDst writes the filter-type byte and Sub-filtered row into dst, which must hold len(row)+1 bytes.
func ApplyFilterSubWithScratchDst(row []byte, bpp int, dst []byte) []byte {
	dst[0] = 1
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		dst[i+1] = row[i] - left
	}
	return dst[:len(row)+1]
}
