package png

func HasAlpha(pixels []byte, colorType ColorType) bool {
	if colorType != ColorRGBA {
		return false
	}

	for i := 3; i < len(pixels); i += 4 {
		if pixels[i] != 255 {
			return true
		}
	}
	return false
}

func OptimizeAlpha(pixels []byte, colorType ColorType) []byte {
	if colorType != ColorRGBA {
		return pixels
	}

	result := make([]byte, len(pixels))
	copy(result, pixels)

	for i := 3; i < len(result); i += 4 {
		if result[i] == 0 {
			result[i-3] = 0
			result[i-2] = 0
			result[i-1] = 0
		}
	}
	return result
}
