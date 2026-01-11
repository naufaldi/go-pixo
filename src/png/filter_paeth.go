package png

func ApplyFilterPaeth(row []byte, prev []byte, bpp int) []byte {
	result := make([]byte, len(row))
	for i := 0; i < len(row); i++ {
		var a, b, c int

		if i >= bpp {
			a = int(row[i-bpp])
		}

		if len(prev) > 0 && i < len(prev) {
			b = int(prev[i])
		}

		if i >= bpp && len(prev) > 0 && i < len(prev) {
			c = int(prev[i-bpp])
		}

		predictor := PaethPredictor(a, b, c)
		result[i] = row[i] - byte(predictor)
	}
	return result
}

func ApplyFilterPaethWithScratch(row []byte, prev []byte, bpp int, scratch *AdaptiveScratch) []byte {
	filtered := scratch.GetFilteredRow()
	filtered[0] = 4
	for i := 0; i < len(row); i++ {
		var a, b, c int

		if i >= bpp {
			a = int(row[i-bpp])
		}

		if len(prev) > 0 && i < len(prev) {
			b = int(prev[i])
		}

		if i >= bpp && len(prev) > 0 && i < len(prev) {
			c = int(prev[i-bpp])
		}

		predictor := PaethPredictor(a, b, c)
		filtered[i+1] = row[i] - byte(predictor)
	}
	return filtered[:len(row)+1]
}

func ApplyFilterPaethWithScratchDst(row []byte, prev []byte, bpp int, dst []byte) []byte {
	dst[0] = 4
	for i := 0; i < len(row); i++ {
		var a, b, c int

		if i >= bpp {
			a = int(row[i-bpp])
		}

		if len(prev) > 0 && i < len(prev) {
			b = int(prev[i])
		}

		if i >= bpp && len(prev) > 0 && i < len(prev) {
			c = int(prev[i-bpp])
		}

		predictor := PaethPredictor(a, b, c)
		dst[i+1] = row[i] - byte(predictor)
	}
	return dst[:len(row)+1]
}
