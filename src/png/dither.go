package png

// DitherStrength represents the intensity of dithering effect.
// Range: 0.0 (no dithering) to 1.0 (maximum dithering).
type DitherStrength float64

// Dithering presets for common use cases.
const (
	DitherNone      DitherStrength = 0.0
	DitherLow       DitherStrength = 0.25
	DitherMedium    DitherStrength = 0.5
	DitherHigh      DitherStrength = 0.75
	DitherMaximum   DitherStrength = 1.0
)

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

// FloydSteinberg applies Floyd-Steinberg dithering with configurable strength.
// Error diffusion reduces visible banding in quantized images.
// strength: 0.0 (no dithering) to 1.0 (maximum dithering effect).
func FloydSteinberg(pixels []byte, palette Palette, strength DitherStrength) []byte {
	return FloydSteinbergWithStrength(pixels, palette, float64(strength))
}

// FloydSteinbergWithStrength applies Floyd-Steinberg dithering with explicit strength value.
func FloydSteinbergWithStrength(pixels []byte, palette Palette, strength float64) []byte {
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

	// Scale factor based on strength
	scale := strength

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

		errR := (r - int(paletteColor.R)) * 7 / 16
		errG := (g - int(paletteColor.G)) * 7 / 16
		errB := (b - int(paletteColor.B)) * 7 / 16

		// Apply strength scaling
		errR = int(float64(errR) * scale)
		errG = int(float64(errG) * scale)
		errB = int(float64(errB) * scale)

		indexed[i] = uint8(paletteIdx)

		if i+1 < width {
			errors[i+1][0] += errR
			errors[i+1][1] += errG
			errors[i+1][2] += errB
		}
		if i+2 < len(errors) {
			errors[i+2][0] += errR / 7
			errors[i+2][1] += errG / 7
			errors[i+2][2] += errB / 7
		}
	}

	return indexed
}

// FloydSteinbergRow applies Floyd-Steinberg dithering row by row with configurable strength.
// This is used for 2D images where errors propagate to the next row.
func FloydSteinbergRow(pixels []byte, palette Palette, prevErrors [][3]int, strength DitherStrength) ([]byte, [][3]int) {
	return FloydSteinbergRowWithStrength(pixels, palette, prevErrors, float64(strength))
}

// FloydSteinbergRowWithStrength applies Floyd-Steinberg dithering row by row with explicit strength.
func FloydSteinbergRowWithStrength(pixels []byte, palette Palette, prevErrors [][3]int, strength float64) ([]byte, [][3]int) {
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

	scale := strength

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

		errR := (r - int(paletteColor.R)) * 3 / 16
		errG := (g - int(paletteColor.G)) * 3 / 16
		errB := (b - int(paletteColor.B)) * 3 / 16

		errR = int(float64(errR) * scale)
		errG = int(float64(errG) * scale)
		errB = int(float64(errB) * scale)

		indexed[i] = uint8(paletteIdx)

		errors[i][0] = errR
		errors[i][1] = errG
		errors[i][2] = errB

		if i+1 < width {
			errors[i+1][0] += errR * 7 / 3
			errors[i+1][1] += errG * 7 / 3
			errors[i+1][2] += errB * 7 / 3
		}
		if i+2 < width {
			errors[i+2][0] += errR * 5 / 3
			errors[i+2][1] += errG * 5 / 3
			errors[i+2][2] += errB * 5 / 3
		}
		if i+1 < len(errors) {
			errors[i+1][0] = clampInt(errors[i+1][0])
			errors[i+1][1] = clampInt(errors[i+1][1])
			errors[i+1][2] = clampInt(errors[i+1][2])
		}
	}

	return indexed, errors
}

// FloydSteinberg2D applies Floyd-Steinberg dithering for 2D images with configurable strength.
// It propagates errors to both right and below pixels.
func FloydSteinberg2D(pixels []byte, width, height int, palette Palette, strength DitherStrength) []byte {
	return FloydSteinberg2DWithStrength(pixels, width, height, palette, float64(strength))
}

// FloydSteinberg2DWithStrength applies Floyd-Steinberg dithering for 2D images with explicit strength.
func FloydSteinberg2DWithStrength(pixels []byte, width, height int, palette Palette, strength float64) []byte {
	bpp := 3 // RGB
	rowSize := width * bpp

	result := make([]byte, width*height)

	var prevErrors [][3]int

	for y := 0; y < height; y++ {
		rowStart := y * rowSize
		rowPixels := pixels[rowStart : rowStart+rowSize]

		indexed, errors := FloydSteinbergRowWithStrength(rowPixels, palette, prevErrors, strength)

		copy(result[y*width:(y+1)*width], indexed)

		prevErrors = errors
	}

	return result
}

// JarvisJudiceNinke applies Jarvis-Judice-Ninke dithering with configurable strength.
// This produces higher quality dithering but is slower.
func JarvisJudiceNinke(pixels []byte, palette Palette, strength DitherStrength) []byte {
	return JarvisJudiceNinkeWithStrength(pixels, palette, float64(strength))
}

// JarvisJudiceNinkeWithStrength applies Jarvis-Judice-Ninke dithering with explicit strength.
func JarvisJudiceNinkeWithStrength(pixels []byte, palette Palette, strength float64) []byte {
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

	scale := strength

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

		errR := (r - int(paletteColor.R)) * 7 / 48
		errG := (g - int(paletteColor.G)) * 7 / 48
		errB := (b - int(paletteColor.B)) * 7 / 48

		errR = int(float64(errR) * scale)
		errG = int(float64(errG) * scale)
		errB = int(float64(errB) * scale)

		indexed[i] = uint8(paletteIdx)

		// Distribute error to neighboring pixels with JJN weights
		// Weights: [right, two-right, three-right, four-right]
		// Using normalized weights: 7/48, 5/48, 3/48, 1/48
		weights := []int{7, 5, 3, 1}
		divisor := 48

		for j, w := range weights {
			if i+j+1 < len(errors) {
				errors[i+j+1][0] += errR * w / divisor * 48 / 7
				errors[i+j+1][1] += errG * w / divisor * 48 / 7
				errors[i+j+1][2] += errB * w / divisor * 48 / 7
			}
		}
	}

	return indexed
}

// Sierra2Row applies Sierra 2-Row dithering, a faster variant of JJN.
// Produces good quality with better performance.
func Sierra2Row(pixels []byte, palette Palette, strength DitherStrength) []byte {
	return Sierra2RowWithStrength(pixels, palette, float64(strength))
}

// Sierra2RowWithStrength applies Sierra 2-Row dithering with explicit strength.
func Sierra2RowWithStrength(pixels []byte, palette Palette, strength float64) []byte {
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
	errors := make([][3]int, width+3)

	scale := strength

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

		errR := (r - int(paletteColor.R)) * 4 / 16
		errG := (g - int(paletteColor.G)) * 4 / 16
		errB := (b - int(paletteColor.B)) * 4 / 16

		errR = int(float64(errR) * scale)
		errG = int(float64(errG) * scale)
		errB = int(float64(errB) * scale)

		indexed[i] = uint8(paletteIdx)

		// Sierra 2-Row error distribution
		if i+1 < width {
			errors[i+1][0] += errR * 2
			errors[i+1][1] += errG * 2
			errors[i+1][2] += errB * 2
		}
		if i+2 < len(errors) {
			errors[i+2][0] += errR
			errors[i+2][1] += errG
			errors[i+2][2] += errB
		}
	}

	return indexed
}

// Stucki applies Stucki dithering, known for clean, professional results.
// Good balance between quality and artifacts.
func Stucki(pixels []byte, palette Palette, strength DitherStrength) []byte {
	return StuckiWithStrength(pixels, palette, float64(strength))
}

// StuckiWithStrength applies Stucki dithering with explicit strength.
func StuckiWithStrength(pixels []byte, palette Palette, strength float64) []byte {
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
	errors := make([][3]int, width+3)

	scale := strength

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

		errR := (r - int(paletteColor.R))
		errG := (g - int(paletteColor.G))
		errB := (b - int(paletteColor.B))

		errR = int(float64(errR) * scale)
		errG = int(float64(errG) * scale)
		errB = int(float64(errB) * scale)

		indexed[i] = uint8(paletteIdx)

		// Stucki error distribution weights
		// Row 1: 8/42, 4/42
		// Row 2: 2/42, 1/42, 1/42, 2/42, 1/42
		if i+1 < width {
			errors[i+1][0] += errR * 8 / 42
			errors[i+1][1] += errG * 8 / 42
			errors[i+1][2] += errB * 8 / 42
		}
		if i+2 < len(errors) {
			errors[i+2][0] += errR * 4 / 42
			errors[i+2][1] += errG * 4 / 42
			errors[i+2][2] += errB * 4 / 42
		}
	}

	return indexed
}

// DitherMethod represents the available dithering algorithms.
type DitherMethod int

const (
	DitherMethodNone DitherMethod = iota
	DitherMethodFloydSteinberg
	DitherMethodJarvisJudiceNinke
	DitherMethodSierra2Row
	DitherMethodStucki
)

// Dither applies dithering using the specified method and strength.
func Dither(pixels []byte, palette Palette, method DitherMethod, strength DitherStrength) []byte {
	switch method {
	case DitherMethodFloydSteinberg:
		return FloydSteinberg(pixels, palette, strength)
	case DitherMethodJarvisJudiceNinke:
		return JarvisJudiceNinke(pixels, palette, strength)
	case DitherMethodSierra2Row:
		return Sierra2Row(pixels, palette, strength)
	case DitherMethodStucki:
		return Stucki(pixels, palette, strength)
	default:
		return Threshold(pixels, palette)
	}
}

// Dither2D applies dithering to 2D images using the specified method and strength.
func Dither2D(pixels []byte, width, height int, palette Palette, method DitherMethod, strength DitherStrength) []byte {
	switch method {
	case DitherMethodFloydSteinberg:
		return FloydSteinberg2D(pixels, width, height, palette, strength)
	default:
		// For other methods, apply row by row
		bpp := 3
		rowSize := width * bpp
		result := make([]byte, width*height)

		var prevErrors [][3]int
		for y := 0; y < height; y++ {
			rowStart := y * rowSize
			rowPixels := pixels[rowStart : rowStart+rowSize]

			var indexed []byte
			indexed, prevErrors = FloydSteinbergRowWithStrength(rowPixels, palette, prevErrors, float64(strength))
			copy(result[y*width:(y+1)*width], indexed)
		}

		return result
	}
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
