package png

// Quantize converts true-color pixels to indexed palette.
// Returns indexed pixels (1 byte per pixel) and palette.
func Quantize(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}

	colorMap := CountColors(pixels, colorType)
	colorsWithCount := ToColorWithCountSlice(colorMap)

	paletteColors := MedianCut(colorsWithCount, maxColors)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	bpp := BytesPerPixel(ColorType(colorType))
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

	return indexed, *palette
}

// QuantizeWithAlpha converts true-color pixels with alpha to indexed palette.
// Returns indexed pixels (1 byte per pixel) and palette with alpha.
func QuantizeWithAlpha(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp

	colorMap := make(map[ColorWithCount]int)
	for i := 0; i < width; i++ {
		offset := i * bpp
		cwc := ColorWithCount{
			Color: Color{
				R: pixels[offset],
				G: pixels[offset+1],
				B: pixels[offset+2],
			},
			Count: 1,
		}
		colorMap[cwc]++
	}

	colorsWithCount := make([]ColorWithCount, 0, len(colorMap))
	for c, count := range colorMap {
		c.Count = count
		colorsWithCount = append(colorsWithCount, c)
	}

	paletteColors := MedianCutWithAlpha(colorsWithCount, maxColors)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

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

	return indexed, *palette
}

// QuantizeToPalette quantizes pixels to a pre-defined palette.
func QuantizeToPalette(pixels []byte, colorType int, palette Palette) []byte {
	bpp := BytesPerPixel(ColorType(colorType))
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

// QuantizeWithDithering applies quantization with Floyd-Steinberg dithering.
func QuantizeWithDithering(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}

	colorMap := CountColors(pixels, colorType)
	colorsWithCount := ToColorWithCountSlice(colorMap)

	paletteColors := MedianCut(colorsWithCount, maxColors)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	bpp := BytesPerPixel(ColorType(colorType))
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
		r := pixelData[i][0] + errors[i][0]
		g := pixelData[i][1] + errors[i][1]
		b := pixelData[i][2] + errors[i][2]

		r = clamp(r)
		g = clamp(g)
		b = clamp(b)

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
		if i+1 < len(errors) {
			errors[i+1][0] = clamp(errors[i+1][0])
			errors[i+1][1] = clamp(errors[i+1][1])
			errors[i+1][2] = clamp(errors[i+1][2])
		}
	}

	return indexed, *palette
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
