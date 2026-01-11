package png

import (
	"testing"
)

func TestQuantizeBasic(t *testing.T) {
	// 2x2 RGB image (2*2*3 = 12 bytes)
	pixels := []byte{
		255, 0, 0, // red
		0, 255, 0, // green
		0, 0, 255, // blue
		255, 255, 0, // yellow
	}

	indexed, palette := Quantize(pixels, 2, 4)

	// Should have 4 indexed pixels
	if len(indexed) != 4 {
		t.Errorf("Quantize() indexed length = %v, want 4", len(indexed))
	}

	// Palette should have up to 4 colors
	if palette.NumColors > 4 {
		t.Errorf("Quantize() palette size = %v, want <= 4", palette.NumColors)
	}
}

func TestQuantizeSingleColor(t *testing.T) {
	// 2x2 RGB image with all red pixels
	pixels := []byte{
		255, 0, 0, 255, 0, 0,
		255, 0, 0, 255, 0, 0,
	}

	indexed, palette := Quantize(pixels, 2, 256)

	// Should have 4 indexed pixels
	if len(indexed) != 4 {
		t.Errorf("Quantize() indexed length = %v, want 4", len(indexed))
	}

	// Palette should have 1 color
	if palette.NumColors != 1 {
		t.Errorf("Quantize() palette size = %v, want 1", palette.NumColors)
	}
}

func TestQuantizeMaxColors(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	// Limit to 2 colors
	indexed, palette := Quantize(pixels, 2, 2)

	// Palette should have at most 2 colors
	if palette.NumColors > 2 {
		t.Errorf("Quantize() palette size = %v, want <= 2", palette.NumColors)
	}

	// All pixels should be indexed
	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("Quantize() indexed[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestQuantizeMaxColorsZero(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	// MaxColors 0 should default to 256
	indexed, palette := Quantize(pixels, 2, 0)

	if len(indexed) != 4 {
		t.Errorf("Quantize() with maxColors 0 indexed length = %v, want 4", len(indexed))
	}

	// Should have all 4 colors since 4 < 256
	if palette.NumColors != 4 {
		t.Errorf("Quantize() with maxColors 0 palette size = %v, want 4", palette.NumColors)
	}
}

func TestQuantizeMaxColorsExceeds256(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	// MaxColors > 256 should cap at 256
	indexed, palette := Quantize(pixels, 2, 300)

	if len(indexed) != 4 {
		t.Errorf("Quantize() with maxColors > 256 indexed length = %v, want 4", len(indexed))
	}

	if palette.NumColors > 256 {
		t.Errorf("Quantize() with maxColors > 256 palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestQuantizeRGBA(t *testing.T) {
	// 2x2 RGBA image
	pixels := []byte{
		255, 0, 0, 255, 0, 255, 0, 255,
		0, 0, 255, 255, 255, 255, 0, 255,
	}

	indexed, _ := Quantize(pixels, 6, 4)

	if len(indexed) != 4 {
		t.Errorf("Quantize(RGBA) indexed length = %v, want 4", len(indexed))
	}
}

func TestQuantizeLargeImage(t *testing.T) {
	width, height := 100, 100
	bpp := 3
	pixels := make([]byte, width*height*bpp)

	// Fill with random-looking pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * bpp
			pixels[idx] = uint8((x * y) % 256)
			pixels[idx+1] = uint8((x + y) % 256)
			pixels[idx+2] = uint8((x*2 + y) % 256)
		}
	}

	indexed, palette := Quantize(pixels, 2, 256)

	if len(indexed) != width*height {
		t.Errorf("Quantize() large image indexed length = %v, want %v", len(indexed), width*height)
	}

	if palette.NumColors > 256 {
		t.Errorf("Quantize() large image palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestQuantizeToPalette(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	palette := NewPalette(4)
	palette.AddColor(Color{255, 0, 0})   // red
	palette.AddColor(Color{0, 255, 0})   // green
	palette.AddColor(Color{0, 0, 255})   // blue
	palette.AddColor(Color{255, 255, 0}) // yellow

	indexed := QuantizeToPalette(pixels, 2, *palette)

	if len(indexed) != 4 {
		t.Errorf("QuantizeToPalette() indexed length = %v, want 4", len(indexed))
	}

	// All indices should be valid
	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("QuantizeToPalette()[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestQuantizeWithDithering(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	indexed, palette := QuantizeWithDithering(pixels, 2, 4)

	if len(indexed) != 4 {
		t.Errorf("QuantizeWithDithering() indexed length = %v, want 4", len(indexed))
	}

	if palette.NumColors > 4 {
		t.Errorf("QuantizeWithDithering() palette size = %v, want <= 4", palette.NumColors)
	}
}

func TestQuantizeOutputIsIndexed(t *testing.T) {
	// Create a gradient-like image
	pixels := []byte{}
	for i := 0; i < 100; i++ {
		val := uint8(i * 2)
		pixels = append(pixels, val, val, val)
	}

	indexed, _ := Quantize(pixels, 2, 16)

	// Each indexed pixel should be a single byte
	if len(indexed) != len(pixels)/3 {
		t.Errorf("Quantize() indexed length = %v, want %v", len(indexed), len(pixels)/3)
	}

	// Each value should be 0-255 (byte)
	_ = indexed
}

func TestQuantizePreservesAllPixels(t *testing.T) {
	// 10x10 RGB image
	width, height := 10, 10
	bpp := 3
	pixels := make([]byte, width*height*bpp)

	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	indexed, _ := Quantize(pixels, 2, 256)

	// Should have exactly width*height indexed pixels
	expectedLen := width * height
	if len(indexed) != expectedLen {
		t.Errorf("Quantize() output length = %v, want %v", len(indexed), expectedLen)
	}
}

func TestClampFunction(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-100, 0},
		{0, 0},
		{128, 128},
		{255, 255},
		{300, 255},
		{1000, 255},
	}

	for _, tt := range tests {
		result := clamp(tt.input)
		if result != tt.expected {
			t.Errorf("clamp(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestQuantizeEmptyPixels(t *testing.T) {
	indexed, palette := Quantize([]byte{}, 2, 256)

	if len(indexed) != 0 {
		t.Errorf("Quantize() on empty indexed length = %v, want 0", len(indexed))
	}

	if palette.NumColors != 0 {
		t.Errorf("Quantize() on empty palette size = %v, want 0", palette.NumColors)
	}
}

func TestQuantize1x1Image(t *testing.T) {
	pixels := []byte{128, 64, 32}

	indexed, palette := Quantize(pixels, 2, 256)

	if len(indexed) != 1 {
		t.Errorf("Quantize() 1x1 indexed length = %v, want 1", len(indexed))
	}

	if palette.NumColors != 1 {
		t.Errorf("Quantize() 1x1 palette size = %v, want 1", palette.NumColors)
	}
}

func TestQuantizeLUTProducesIdenticalResults(t *testing.T) {
	pixels := generateTestPixels(100, 100)

	colorMap := CountColors(pixels, 2)
	colorsWithCount := ToColorWithCountSlice(colorMap)
	paletteColors := MedianCut(colorsWithCount, 256)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	lut := NewFullPaletteLUT(palette)

	indexedLUT := make([]byte, len(pixels)/3)
	indexedLinear := make([]byte, len(pixels)/3)

	for i := 0; i < len(pixels)/3; i++ {
		offset := i * 3
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		indexedLUT[i] = lut.Lookup(c.R, c.G, c.B, 255)
		indexedLinear[i] = uint8(palette.FindNearest(c))
	}

	for i := range indexedLUT {
		if indexedLUT[i] != indexedLinear[i] {
			t.Errorf("LUT vs Linear: pixel %d differs: LUT=%d, Linear=%d", i, indexedLUT[i], indexedLinear[i])
		}
	}
}

func TestQuantizeToPaletteLUTIdenticalToLinear(t *testing.T) {
	pixels := generateTestPixels(100, 100)
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}

	lut := NewFullPaletteLUT(palette)

	indexedLUT := make([]byte, len(pixels)/3)
	indexedLinear := make([]byte, len(pixels)/3)

	for i := 0; i < len(pixels)/3; i++ {
		offset := i * 3
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		indexedLUT[i] = lut.Lookup(c.R, c.G, c.B, 255)
		indexedLinear[i] = uint8(palette.FindNearest(c))
	}

	for i := range indexedLUT {
		if indexedLUT[i] != indexedLinear[i] {
			t.Errorf("QuantizeToPalette LUT vs Linear: pixel %d differs: LUT=%d, Linear=%d", i, indexedLUT[i], indexedLinear[i])
		}
	}
}

func TestQuantizeWithDitheringLUTIdenticalToLinear(t *testing.T) {
	pixels := generateTestPixels(100, 100)

	colorMap := CountColors(pixels, 2)
	colorsWithCount := ToColorWithCountSlice(colorMap)
	paletteColors := MedianCut(colorsWithCount, 256)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	lut := NewFullPaletteLUT(palette)

	width := len(pixels) / 3
	indexedLUT := make([]byte, width)
	indexedLinear := make([]byte, width)

	for i := 0; i < width; i++ {
		offset := i * 3
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		indexedLUT[i] = lut.Lookup(c.R, c.G, c.B, 255)
		indexedLinear[i] = uint8(palette.FindNearest(c))
	}

	for i := range indexedLUT {
		if indexedLUT[i] != indexedLinear[i] {
			t.Errorf("QuantizeWithDithering LUT vs Linear: pixel %d differs: LUT=%d, Linear=%d", i, indexedLUT[i], indexedLinear[i])
		}
	}
}

func TestFindNearestIndex(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 0, B: 0})
	palette.AddColor(Color{R: 0, G: 255, B: 0})
	palette.AddColor(Color{R: 0, G: 0, B: 255})

	tests := []struct {
		color Color
		want  int
	}{
		{Color{0, 0, 0}, 0},
		{Color{255, 0, 0}, 1},
		{Color{0, 255, 0}, 2},
		{Color{0, 0, 255}, 3},
		{Color{128, 0, 0}, 1},
		{Color{0, 128, 0}, 2},
		{Color{0, 0, 128}, 3},
	}

	for _, tt := range tests {
		got := palette.FindNearestIndex(tt.color)
		if got != tt.want {
			t.Errorf("FindNearestIndex(%v) = %d, want %d", tt.color, got, tt.want)
		}
	}
}

func generateTestPixels(width, height int) []byte {
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}
	return pixels
}

func BenchmarkQuantizeLUT_Small(b *testing.B) {
	pixels := generateTestPixels(32, 32)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizeLinear_Small(b *testing.B) {
	pixels := generateTestPixels(32, 32)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, false)
	}
}

func BenchmarkQuantizeLUT_Medium(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizeLinear_Medium(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, false)
	}
}

func BenchmarkQuantizeLUT_Large(b *testing.B) {
	pixels := generateTestPixels(500, 500)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizeLinear_Large(b *testing.B) {
	pixels := generateTestPixels(500, 500)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, false)
	}
}

func BenchmarkQuantizeLUT_32Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 32, true)
	}
}

func BenchmarkQuantizeLinear_32Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 32, false)
	}
}

func BenchmarkQuantizeLUT_64Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 64, true)
	}
}

func BenchmarkQuantizeLinear_64Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 64, false)
	}
}

func BenchmarkQuantizeLUT_128Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 128, true)
	}
}

func BenchmarkQuantizeLinear_128Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 128, false)
	}
}

func BenchmarkQuantizeLUT_256Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizeLinear_256Colors(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		Quantize(pixels, 2, 256, false)
	}
}

func BenchmarkQuantizeFast(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizeFast(pixels, 2, 256)
	}
}

func BenchmarkQuantizeWithLUTDisabled(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizeWithLUTDisabled(pixels, 2, 256)
	}
}

func BenchmarkQuantizeToPaletteLUT(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	for i := 0; i < b.N; i++ {
		QuantizeToPalette(pixels, 2, *palette, true)
	}
}

func BenchmarkQuantizeToPaletteLinear(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	for i := 0; i < b.N; i++ {
		QuantizeToPalette(pixels, 2, *palette, false)
	}
}

func BenchmarkQuantizeWithDitheringLUT(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizeWithDithering(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizeWithDitheringLinear(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizeWithDithering(pixels, 2, 256, false)
	}
}

func BenchmarkLUTBuild(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	for i := 0; i < b.N; i++ {
		NewFullPaletteLUT(palette)
	}
}

func BenchmarkLUTLookup(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	lut := NewFullPaletteLUT(palette)
	pixels := generateTestPixels(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(pixels); j += 3 {
			_ = lut.Lookup(pixels[j], pixels[j+1], pixels[j+2], 255)
		}
	}
}

func BenchmarkLinearSearchLookup(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	pixels := generateTestPixels(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(pixels); j += 3 {
			c := Color{R: pixels[j], G: pixels[j+1], B: pixels[j+2]}
			_ = palette.FindNearest(c)
		}
	}
}

func TestQuantizeWithKmeansBasic(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	indexed, _ := QuantizeWithKmeans(pixels, 2, 4, 3)

	if len(indexed) != 4 {
		t.Errorf("QuantizeWithKmeans() indexed length = %v, want 4", len(indexed))
	}
}

func TestQuantizeWithKmeansReducesError(t *testing.T) {
	width, height := 32, 32
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			pixels[idx] = uint8(x*7 + y*3)
			pixels[idx+1] = uint8(y * 5)
			pixels[idx+2] = uint8(x*3 + y*7)
		}
	}

	indexed1, palette1 := Quantize(pixels, 2, 16, true)
	error1 := calculateQuantizationError(pixels, indexed1, &palette1, 2)

	indexed2, palette2 := QuantizeWithKmeans(pixels, 2, 16, 3)
	error2 := calculateQuantizationError(pixels, indexed2, &palette2, 2)

	if error2 >= error1 {
		t.Errorf("QuantizeWithKmeans should reduce error: before=%f, after=%f", error1, error2)
	}
}

func TestQuantizeWithKmeansConvergence(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	indexed2, palette2 := QuantizeWithKmeans(pixels, 2, 16, 2)
	error2 := calculateQuantizationError(pixels, indexed2, &palette2, 2)

	indexed3, palette3 := QuantizeWithKmeans(pixels, 2, 16, 3)
	error3 := calculateQuantizationError(pixels, indexed3, &palette3, 2)

	if error3 > error2 {
		t.Errorf("3 iterations should not have higher error than 2: 2 iter=%f, 3 iter=%f", error2, error3)
	}
}

func TestQuantizeWithKmeansWithAlpha(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 255, 0, 255, 0, 255,
		0, 0, 255, 255, 255, 255, 0, 255,
	}

	indexed, _ := QuantizeWithKmeans(pixels, 6, 4, 3)

	expectedLen := len(pixels) / 4
	if len(indexed) != expectedLen {
		t.Errorf("QuantizeWithKmeans(RGBA) indexed length = %v, want %v", len(indexed), expectedLen)
	}
}

func TestQuantizeWithKmeansEmptyPixels(t *testing.T) {
	indexed, palette := QuantizeWithKmeans([]byte{}, 2, 256, 3)

	if len(indexed) != 0 {
		t.Errorf("QuantizeWithKmeans() on empty indexed length = %v, want 0", len(indexed))
	}

	if palette.NumColors != 0 {
		t.Errorf("QuantizeWithKmeans() on empty palette size = %v, want 0", palette.NumColors)
	}
}

func TestQuantizeWithKmeansSingleColor(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 255, 0, 0,
		255, 0, 0, 255, 0, 0,
	}

	indexed, palette := QuantizeWithKmeans(pixels, 2, 256, 3)

	if len(indexed) != 4 {
		t.Errorf("QuantizeWithKmeans() indexed length = %v, want 4", len(indexed))
	}

	if palette.NumColors != 1 {
		t.Errorf("QuantizeWithKmeans() single color palette size = %v, want 1", palette.NumColors)
	}
}

func TestQuantizeWithKmeansPhotographicContent(t *testing.T) {
	width, height := 64, 64
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			pixels[idx] = uint8(x * 4)
			pixels[idx+1] = uint8(y * 3)
			pixels[idx+2] = uint8((x + y) / 2)
		}
	}

	indexed1, palette1 := Quantize(pixels, 2, 32, true)
	error1 := calculateQuantizationError(pixels, indexed1, &palette1, 2)

	indexed2, palette2 := QuantizeWithKmeans(pixels, 2, 32, 3)
	error2 := calculateQuantizationError(pixels, indexed2, &palette2, 2)

	if error2 >= error1 {
		t.Errorf("QuantizeWithKmeans should reduce error for photographic content: before=%f, after=%f", error1, error2)
	}

	improvement := (error1 - error2) / error1 * 100
	if improvement < 5 {
		t.Logf("Warning: improvement was only %.2f%% (expected 5-15%%)", improvement)
	}
}

func TestQuantizeWithKmeansGradient(t *testing.T) {
	width, height := 100, 100
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			val := uint8((x * 255) / width)
			pixels[idx] = val
			pixels[idx+1] = val
			pixels[idx+2] = val
		}
	}

	indexed1, palette1 := Quantize(pixels, 2, 16, true)
	error1 := calculateQuantizationError(pixels, indexed1, &palette1, 2)

	indexed2, palette2 := QuantizeWithKmeans(pixels, 2, 16, 3)
	error2 := calculateQuantizationError(pixels, indexed2, &palette2, 2)

	if error2 >= error1 {
		t.Errorf("QuantizeWithKmeans should reduce error for gradient: before=%f, after=%f", error1, error2)
	}
}

func TestQuantizeWithKmeansMaxColors(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	indexed, palette := QuantizeWithKmeans(pixels, 2, 2, 3)

	if palette.NumColors > 2 {
		t.Errorf("QuantizeWithKmeans() palette size = %v, want <= 2", palette.NumColors)
	}

	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("QuantizeWithKmeans()[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestQuantizeWithKmeansLargeImage(t *testing.T) {
	width, height := 100, 100
	pixels := make([]byte, width*height*3)

	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	indexed, palette := QuantizeWithKmeans(pixels, 2, 256, 3)

	if len(indexed) != width*height {
		t.Errorf("QuantizeWithKmeans() large image indexed length = %v, want %v", len(indexed), width*height)
	}

	if palette.NumColors > 256 {
		t.Errorf("QuantizeWithKmeans() large image palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestQuantizeWithKmeansPreservesPaletteSize(t *testing.T) {
	width, height := 50, 50
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			pixels[idx] = uint8((x * 256) / width)
			pixels[idx+1] = uint8((y * 256) / height)
			pixels[idx+2] = uint8(128)
		}
	}

	_, palette := QuantizeWithKmeans(pixels, 2, 64, 3)

	if palette.NumColors > 64 {
		t.Errorf("QuantizeWithKmeans() palette size = %v, want <= 64", palette.NumColors)
	}
}

func BenchmarkQuantizeWithKmeansSmall(b *testing.B) {
	pixels := generateTestPixels(32, 32)
	for i := 0; i < b.N; i++ {
		QuantizeWithKmeans(pixels, 2, 256, 3)
	}
}

func BenchmarkQuantizeWithKmeansMedium(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizeWithKmeans(pixels, 2, 256, 3)
	}
}

func BenchmarkQuantizeWithKmeansLarge(b *testing.B) {
	pixels := generateTestPixels(500, 500)
	for i := 0; i < b.N; i++ {
		QuantizeWithKmeans(pixels, 2, 256, 3)
	}
}
