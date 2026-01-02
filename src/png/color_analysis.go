package png

func IsGrayscale(pixels []byte, colorType ColorType) bool {
	switch colorType {
	case ColorGrayscale:
		return true
	case ColorRGB:
		return isGrayscaleRGB(pixels)
	case ColorRGBA:
		return isGrayscaleRGBA(pixels)
	default:
		return false
	}
}

func isGrayscaleRGB(pixels []byte) bool {
	if len(pixels) == 0 {
		return true
	}

	for i := 0; i < len(pixels); i += 3 {
		if pixels[i] != pixels[i+1] || pixels[i+1] != pixels[i+2] {
			return false
		}
	}
	return true
}

func isGrayscaleRGBA(pixels []byte) bool {
	if len(pixels) == 0 {
		return true
	}

	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] != pixels[i+1] || pixels[i+1] != pixels[i+2] {
			return false
		}
	}
	return true
}

func CanReduceToGrayscale(pixels []byte, width, height int, colorType ColorType) bool {
	bpp := BytesPerPixel(colorType)
	expectedLen := width * height * bpp
	if len(pixels) != expectedLen {
		return false
	}

	return IsGrayscale(pixels, colorType)
}

func CanReduceToRGB(pixels []byte, width, height int) bool {
	if len(pixels) != width*height*4 {
		return false
	}

	for i := 3; i < len(pixels); i += 4 {
		if pixels[i] != 255 {
			return false
		}
	}
	return true
}
