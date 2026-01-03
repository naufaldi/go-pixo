package png

// Threshold applies no dithering, direct palette mapping.
// Each pixel is simply mapped to the nearest palette color.
func Threshold(pixels []byte, palette Palette) []byte {
	bpp := 3 // RGB
	width := len(pixels) / bpp

	indexed := make([]byte, width)

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		indexed[i] = uint8(palette.FindNearest(c))
	}

	return indexed
}

// FloydSteinberg applies Floyd-Steinberg dithering.
// Error diffusion reduces visible banding in quantized images.
func FloydSteinberg(pixels []byte, palette Palette) []byte {
	bpp := 3 // RGB
	width := len(pixels) / bpp

	pixelData := make([][3]int, width)
	for i := 0; i < width; i++ {
		offset := i * bpp
		pixelData[i] = [3]int{
			int(pixels[offset]),
			int(pixels[offset+1]),
			int(pixels[offset+2]),
		}
	}

	indexed := make([]byte, width)
	errors := make([][3]int, width+2)

	for i := 0; i < width; i++ {
		r := clampInt(pixelData[i][0] + errors[i][0])
		g := clampInt(pixelData[i][1] + errors[i][1])
		b := clampInt(pixelData[i][2] + errors[i][2])

		c := Color{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
		}

		paletteIdx := palette.FindNearest(c)
		paletteColor := palette.Colors[paletteIdx]

		errR := r - int(paletteColor.R)
		errG := g - int(paletteColor.G)
		errB := b - int(paletteColor.B)

		indexed[i] = uint8(paletteIdx)

		if i+1 < width {
			errors[i+1][0] += errR * 7 / 16
			errors[i+1][1] += errG * 7 / 16
			errors[i+1][2] += errB * 7 / 16
		}
		if i+2 < len(errors) {
			errors[i+2][0] += errR * 1 / 16
			errors[i+2][1] += errG * 1 / 16
			errors[i+2][2] += errB * 1 / 16
		}
	}

	return indexed
}

// FloydSteinbergRow applies Floyd-Steinberg dithering row by row.
// This is used for 2D images where errors propagate to the next row.
func FloydSteinbergRow(pixels []byte, palette Palette, prevErrors [][3]int) ([]byte, [][3]int) {
	bpp := 3 // RGB
	width := len(pixels) / bpp

	pixelData := make([][3]int, width)
	for i := 0; i < width; i++ {
		offset := i * bpp
		pixelData[i] = [3]int{
			int(pixels[offset]),
			int(pixels[offset+1]),
			int(pixels[offset+2]),
		}
	}

	indexed := make([]byte, width)
	errors := make([][3]int, width+2)

	for i := 0; i < width; i++ {
		r := pixelData[i][0]
		g := pixelData[i][1]
		b := pixelData[i][2]

		if prevErrors != nil && i < len(prevErrors) {
			r += prevErrors[i][0]
			g += prevErrors[i][1]
			b += prevErrors[i][2]
		}

		r = clampInt(r)
		g = clampInt(g)
		b = clampInt(b)

		c := Color{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
		}

		paletteIdx := palette.FindNearest(c)
		paletteColor := palette.Colors[paletteIdx]

		errR := r - int(paletteColor.R)
		errG := g - int(paletteColor.G)
		errB := b - int(paletteColor.B)

		indexed[i] = uint8(paletteIdx)

		errors[i][0] = errR * 3 / 16
		errors[i][1] = errG * 3 / 16
		errors[i][2] = errB * 3 / 16

		if i+1 < width {
			errors[i+1][0] += errR * 7 / 16
			errors[i+1][1] += errG * 7 / 16
			errors[i+1][2] += errB * 7 / 16
		}
		if i+2 < width {
			errors[i+2][0] += errR * 5 / 16
			errors[i+2][1] += errG * 5 / 16
			errors[i+2][2] += errB * 5 / 16
		}
		if i+1 < len(errors) {
			errors[i+1][0] = clampInt(errors[i+1][0])
			errors[i+1][1] = clampInt(errors[i+1][1])
			errors[i+1][2] = clampInt(errors[i+1][2])
		}
	}

	return indexed, errors
}

// FloydSteinberg2D applies Floyd-Steinberg dithering for 2D images.
// It propagates errors to both right and below pixels.
func FloydSteinberg2D(pixels []byte, width, height int, palette Palette) []byte {
	bpp := 3 // RGB
	rowSize := width * bpp

	result := make([]byte, width*height)

	var prevErrors [][3]int

	for y := 0; y < height; y++ {
		rowStart := y * rowSize
		rowPixels := pixels[rowStart : rowStart+rowSize]

		indexed, errors := FloydSteinbergRow(rowPixels, palette, prevErrors)

		copy(result[y*width:(y+1)*width], indexed)

		prevErrors = errors
	}

	return result
}

// JarvisJudiceNinke applies Jarvis-Judice-Ninke dithering.
// This produces higher quality dithering but is slower.
func JarvisJudiceNinke(pixels []byte, palette Palette) []byte {
	bpp := 3 // RGB
	width := len(pixels) / bpp

	pixelData := make([][3]int, width)
	for i := 0; i < width; i++ {
		offset := i * bpp
		pixelData[i] = [3]int{
			int(pixels[offset]),
			int(pixels[offset+1]),
			int(pixels[offset+2]),
		}
	}

	indexed := make([]byte, width)
	errors := make([][3]int, width+4)

	for i := 0; i < width; i++ {
		r := clampInt(pixelData[i][0] + errors[i][0])
		g := clampInt(pixelData[i][1] + errors[i][1])
		b := clampInt(pixelData[i][2] + errors[i][2])

		c := Color{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
		}

		paletteIdx := palette.FindNearest(c)
		paletteColor := palette.Colors[paletteIdx]

		errR := r - int(paletteColor.R)
		errG := g - int(paletteColor.G)
		errB := b - int(paletteColor.B)

		indexed[i] = uint8(paletteIdx)

		// Distribute error to neighboring pixels
		weights := [][3]int{
			{7, 0, 0}, // i+1
			{5, 0, 0}, // i+2
			{3, 0, 0}, // i+3
			{1, 0, 0}, // i+4
		}
		divisors := []int{48, 48, 48, 48}

		for j, w := range weights {
			if i+j+1 < len(errors) {
				errors[i+j+1][0] += errR * w[0] / divisors[j]
				errors[i+j+1][1] += errG * w[1] / divisors[j]
				errors[i+j+1][2] += errB * w[2] / divisors[j]
			}
		}
	}

	return indexed
}

func clampInt(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
