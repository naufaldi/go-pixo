package png

// SumAbsoluteValues returns the sum of the absolute values of the bytes in filtered (treating bytes as signed).
func SumAbsoluteValues(filtered []byte) int {
	sum := 0
	for _, b := range filtered {
		signed := int(int8(b))
		if signed < 0 {
			sum -= signed
		} else {
			sum += signed
		}
	}
	return sum
}
