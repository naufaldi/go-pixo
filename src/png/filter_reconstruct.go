package png

func ReconstructNone(filtered []byte) []byte {
	result := make([]byte, len(filtered))
	copy(result, filtered)
	return result
}

func ReconstructSub(filtered []byte, bpp int) []byte {
	result := make([]byte, len(filtered))
	for i := 0; i < len(filtered); i++ {
		var left byte
		if i >= bpp {
			left = result[i-bpp]
		}
		result[i] = filtered[i] + left
	}
	return result
}

func ReconstructUp(filtered []byte, prev []byte) []byte {
	result := make([]byte, len(filtered))
	for i := 0; i < len(filtered); i++ {
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		result[i] = filtered[i] + up
	}
	return result
}

func ReconstructAverage(filtered []byte, prev []byte, bpp int) []byte {
	result := make([]byte, len(filtered))
	for i := 0; i < len(filtered); i++ {
		var left byte
		if i >= bpp {
			left = result[i-bpp]
		}
		var up byte
		if len(prev) > 0 && i < len(prev) {
			up = prev[i]
		}
		avg := (uint16(left) + uint16(up)) / 2
		result[i] = filtered[i] + byte(avg)
	}
	return result
}

func ReconstructPaeth(filtered []byte, prev []byte, bpp int) []byte {
	result := make([]byte, len(filtered))
	for i := 0; i < len(filtered); i++ {
		var a, b, c int

		if i >= bpp {
			a = int(result[i-bpp])
		}

		if len(prev) > 0 && i < len(prev) {
			b = int(prev[i])
		}

		if i >= bpp && len(prev) > 0 && i < len(prev) {
			c = int(prev[i-bpp])
		}

		predictor := PaethPredictor(a, b, c)
		result[i] = filtered[i] + byte(predictor)
	}
	return result
}
