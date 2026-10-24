package png

// Small jobs avoid cache allocation and palette copying entirely.
const paletteCacheMinPixels = 1024

// QuantizeWithKmeans converts true-color pixels to indexed palette using K-means refinement.
// This function provides 5-15% improvement in visual quality for photographic content.
// Returns indexed pixels (1 byte per pixel) and refined palette.
// Parameters:
//   - pixels: input true-color pixel data
//   - colorType: PNG color type (2 for RGB, 6 for RGBA)
//   - maxColors: maximum number of palette colors (1-256)
//   - iterations: number of K-means refinement iterations (recommended: 2-3)
func QuantizeWithKmeans(pixels []byte, colorType int, maxColors int, iterations int) ([]byte, Palette) {
	if iterations <= 0 {
		iterations = 2
	}

	indexed, palette := Quantize(pixels, colorType, maxColors, true)

	if palette.NumColors == 0 {
		return indexed, palette
	}

	colorMap := CountColors(pixels, colorType)
	colorCountSlice := ToColorCount(colorMap)

	refinedColors := RefinePaletteKmeans(&palette, colorCountSlice, iterations)

	for i := 0; i < len(refinedColors) && i < palette.NumColors; i++ {
		palette.Colors[i] = refinedColors[i]
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp

	var lut *PaletteLUT
	if width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewFullPaletteLUT(&palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{R: pixels[offset], G: pixels[offset+1], B: pixels[offset+2]}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearest(c))
		}
	}

	return indexed, palette
}

// Quantize converts true-color pixels to indexed palette.
// Returns indexed pixels (1 byte per pixel) and palette.
// If useLUT is true (default), sufficiently large jobs memoize exact RGB searches.
// The cache is bounded; misses use linear search with identical results.
// Set useLUT to false to use linear search (for testing/benchmarking).
func Quantize(pixels []byte, colorType int, maxColors int, useLUT ...bool) ([]byte, Palette) {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

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

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewFullPaletteLUT(palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearest(c))
		}
	}

	return indexed, *palette
}

// QuantizeFast is equivalent to Quantize with bounded memoization enabled.
func QuantizeFast(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	return Quantize(pixels, colorType, maxColors, true)
}

// QuantizeWithLUTDisabled is equivalent to Quantize but always uses linear search.
// This is useful for testing and benchmarking comparison.
func QuantizeWithLUTDisabled(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	return Quantize(pixels, colorType, maxColors, false)
}

// QuantizeWithAlpha converts true-color pixels with alpha to indexed palette.
// Returns indexed pixels (1 byte per pixel) and palette with alpha.
func QuantizeWithAlpha(pixels []byte, colorType int, maxColors int, useLUT ...bool) ([]byte, Palette) {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

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

	var lut *PaletteLUT
	if enableLUT && palette.NumColors > 0 {
		lut = NewPaletteLUT(palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearest(c))
		}
	}

	return indexed, *palette
}

// QuantizeToPalette quantizes pixels to a pre-defined palette.
func QuantizeToPalette(pixels []byte, colorType int, palette Palette, useLUT ...bool) []byte {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp

	indexed := make([]byte, width)

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewFullPaletteLUT(&palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearest(c))
		}
	}

	return indexed
}

// QuantizeWithDithering preserves the legacy one-dimensional 7/16 error carry.
// It retains only the next pixel error, not an image-sized diffusion buffer.
func QuantizeWithDithering(pixels []byte, colorType int, maxColors int, useLUT ...bool) ([]byte, Palette) {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

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
	var carriedError [3]int

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewFullPaletteLUT(palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		r := int(pixels[offset]) + carriedError[0]
		g := int(pixels[offset+1]) + carriedError[1]
		b := int(pixels[offset+2]) + carriedError[2]

		r = clamp(r)
		g = clamp(g)
		b = clamp(b)

		c := Color{
			R: uint8(r),
			G: uint8(g),
			B: uint8(b),
		}

		var paletteIdx int
		if lut != nil {
			paletteIdx = int(lut.Lookup(c.R, c.G, c.B, 255))
		} else {
			paletteIdx = palette.FindNearest(c)
		}
		paletteColor := palette.Colors[paletteIdx]

		errR := r - int(paletteColor.R)
		errG := g - int(paletteColor.G)
		errB := b - int(paletteColor.B)

		indexed[i] = uint8(paletteIdx)

		// Preserve the legacy positive-only carry to the next linear pixel.
		// This helper has no row width; row-aware diffusion is a separate change.
		carriedError = [3]int{clamp(errR * 7 / 16), clamp(errG * 7 / 16), clamp(errB * 7 / 16)}
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

func calculateQuantizationError(pixels []byte, indexed []byte, palette *Palette, colorType int) float64 {
	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp
	var totalError float64

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		paletteColor := palette.Colors[indexed[i]]
		dr := float64(c.R) - float64(paletteColor.R)
		dg := float64(c.G) - float64(paletteColor.G)
		db := float64(c.B) - float64(paletteColor.B)
		totalError += dr*dr + dg*dg + db*db
	}

	return totalError
}

func calculatePerceptualQuantizationError(pixels []byte, indexed []byte, palette *Palette, colorType int) float64 {
	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp
	var totalError float64

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		paletteColor := palette.Colors[indexed[i]]
		totalError += RedmeanDistanceSq(c, paletteColor)
	}

	return totalError
}

// QuantizePerceptual converts true-color pixels to indexed palette using perceptual distance metric.
// Uses Redmean distance which provides better visual quality for skin tones and gradients.
// Returns indexed pixels (1 byte per pixel) and palette.
// If useLUT is true (default), sufficiently large jobs memoize exact searches.
// Set useLUT to false to always use linear search.
func QuantizePerceptual(pixels []byte, colorType int, maxColors int, useLUT ...bool) ([]byte, Palette) {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

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

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewPerceptualPaletteLUT(palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearestPerceptual(c))
		}
	}

	return indexed, *palette
}

// QuantizeFastPerceptual enables bounded memoization for QuantizePerceptual.
func QuantizeFastPerceptual(pixels []byte, colorType int, maxColors int) ([]byte, Palette) {
	return QuantizePerceptual(pixels, colorType, maxColors, true)
}

// QuantizeToPalettePerceptual quantizes pixels to a pre-defined palette using perceptual distance.
func QuantizeToPalettePerceptual(pixels []byte, colorType int, palette Palette, useLUT ...bool) []byte {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp

	indexed := make([]byte, width)

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		lut = NewPerceptualPaletteLUT(&palette)
	}

	for i := 0; i < width; i++ {
		offset := i * bpp
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else {
			indexed[i] = uint8(palette.FindNearestPerceptual(c))
		}
	}

	return indexed
}

// QuantizeWithOptions converts true-color pixels to indexed palette using Options.
// QualityTarget controls the existing median-cut heuristic, not a measured quality
// floor. Distance settings apply to every lookup, including error-adjusted pixels.
// Dithering uses only horizontal 7/16 diffusion, reset at each Options.Width row.
// Passing useLUT=false uses linear searches instead of bounded memoization.
func QuantizeWithOptions(pixels []byte, colorType int, maxColors int, opts Options, useLUT ...bool) ([]byte, Palette) {
	enableLUT := true
	if len(useLUT) > 0 {
		enableLUT = useLUT[0]
	}

	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}

	colorMap := countColorsWithProgress(pixels, colorType, opts.ProgressCallback)
	// Sorting has no useful counter. Report its boundary, without claiming work units.
	if opts.ProgressCallback != nil {
		opts.ProgressCallback("palette-sort", 0)
	}
	colorsWithCount := ToColorWithCountSlice(colorMap)

	quality := opts.QualityTarget
	if quality == 0 && !opts.QualityTargetSet {
		quality = 100
	}
	// The median-cut quality heuristic has a minimum of two buckets. A caller's
	// explicit one-color cap takes precedence over that heuristic.
	if maxColors == 1 {
		quality = 100
	}
	if quality < 0 {
		quality = 0
	}
	if quality > 100 {
		quality = 100
	}
	paletteColors := medianCutWithProgress(colorsWithCount, maxColors, float64(quality)/100, opts.ProgressCallback)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(pixels) / bpp

	indexed := make([]byte, width)

	usePerceptual := opts.UsePerceptualDistance || opts.DistanceMetric == DistanceMetricRedmean
	strength := opts.DitheringStrength
	if strength < 0 {
		strength = 0
	}
	if strength > 1 {
		strength = 1
	}
	var carry [3]int

	var lut *PaletteLUT
	if enableLUT && width >= paletteCacheMinPixels && palette.NumColors > 1 {
		if usePerceptual {
			lut = NewPerceptualPaletteLUT(palette)
		} else {
			lut = NewFullPaletteLUT(palette)
		}
	}

	progress := newWorkProgress(opts.ProgressCallback, "quantize", width, 4096)
	for i := 0; i < width; i++ {
		offset := i * bpp
		if opts.Width > 0 && i%opts.Width == 0 {
			carry = [3]int{}
		}
		c := Color{
			R: uint8(clamp(int(pixels[offset]) + carry[0])),
			G: uint8(clamp(int(pixels[offset+1]) + carry[1])),
			B: uint8(clamp(int(pixels[offset+2]) + carry[2])),
		}
		if lut != nil {
			indexed[i] = lut.Lookup(c.R, c.G, c.B, 255)
		} else if usePerceptual {
			indexed[i] = uint8(palette.FindNearestPerceptual(c))
		} else {
			indexed[i] = uint8(palette.FindNearest(c))
		}
		if opts.Dithering && strength > 0 && palette.NumColors > 0 {
			chosen := palette.Colors[indexed[i]]
			carry = [3]int{
				int(float64(int(c.R)-int(chosen.R)) * 7 / 16 * strength),
				int(float64(int(c.G)-int(chosen.G)) * 7 / 16 * strength),
				int(float64(int(c.B)-int(chosen.B)) * 7 / 16 * strength),
			}
		}
		if i+1 >= progress.next {
			progress.report(i + 1)
		}
	}
	progress.finish()

	return indexed, *palette
}
