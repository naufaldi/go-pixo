package png

func ApplyFilterAverage(row []byte, prev []byte, bpp int) []byte {
	result := make([]byte, len(row))
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		avg := (uint16(left) + uint16(up)) / 2
		result[i] = row[i] - byte(avg)
	}
	return result
}

func ApplyFilterAverageWithScratch(row []byte, prev []byte, bpp int, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 3
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		avg := (uint16(left) + uint16(up)) / 2
		filtered[i+1] = row[i] - byte(avg)
	}
	return filtered[:len(row)+1]
}

func ApplyFilterAverageWithScratchDst(row []byte, prev []byte, bpp int, dst []byte) []byte {
	dst[0] = 3
	for i := 0; i < len(row); i++ {
		var left byte
		if i >= bpp {
			left = row[i-bpp]
		}
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		avg := (uint16(left) + uint16(up)) / 2
		dst[i+1] = row[i] - byte(avg)
	}
	return dst[:len(row)+1]
}
