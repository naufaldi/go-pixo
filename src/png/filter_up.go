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
