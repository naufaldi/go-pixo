package png

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
