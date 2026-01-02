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
