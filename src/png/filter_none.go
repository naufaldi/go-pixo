package png

func ApplyFilterNone(row []byte) []byte {
	result := make([]byte, len(row))
	copy(result, row)
	return result
}
