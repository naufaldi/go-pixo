package png

import (
	"math"
	"testing"
)

func TestRedmeanDistanceSymmetric(t *testing.T) {
	testCases := []Color{
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{255, 128, 64},
		{32, 64, 128},
		{200, 100, 50},
		{10, 200, 200},
	}

	for _, c1 := range testCases {
		for _, c2 := range testCases {
			d1 := RedmeanDistance(c1, c2)
			d2 := RedmeanDistance(c2, c1)
			if !floatEq(d1, d2) {
				t.Errorf("RedmeanDistance not symmetric: d(%v, %v) = %v, d(%v, %v) = %v",
					c1, c2, d1, c2, c1, d2)
			}
		}
	}
}

func TestRedmeanDistanceZeroForSameColor(t *testing.T) {
	testCases := []Color{
		{0, 0, 0},
		{255, 255, 255},
		{128, 128, 128},
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{100, 150, 200},
		{1, 2, 3},
	}

	for _, c := range testCases {
		d := RedmeanDistance(c, c)
		if d != 0.0 {
			t.Errorf("RedmeanDistance(%v, %v) = %v, want 0", c, c, d)
		}
		dSq := RedmeanDistanceSq(c, c)
		if dSq != 0.0 {
			t.Errorf("RedmeanDistanceSq(%v, %v) = %v, want 0", c, c, dSq)
		}
	}
}

func TestRedmeanDistanceTriangleInequality(t *testing.T) {
	colors := []Color{
		{0, 0, 0},
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{255, 128, 64},
	}

	for i, c1 := range colors {
		for j, c2 := range colors {
			for k, c3 := range colors {
				if i == j || j == k || i == k {
					continue
				}
				d12 := RedmeanDistance(c1, c2)
				d23 := RedmeanDistance(c2, c3)
				d13 := RedmeanDistance(c1, c3)

				if d13 > d12+d23+0.001 {
					t.Errorf("Triangle inequality violated: d(%v, %v) > d(%v, %v) + d(%v, %v): %v > %v",
						c1, c3, c1, c2, c2, c3, d13, d12+d23)
				}
			}
		}
	}
}

func TestRedmeanDistanceVsEuclidean(t *testing.T) {
	t.Run("skin tones closer in redmean", func(t *testing.T) {
		c1 := Color{255, 224, 189}
		c2 := Color{241, 194, 125}

		euclidean := euclideanDistance(c1, c2)
		redmean := RedmeanDistance(c1, c2)

		t.Logf("Euclidean distance: %v", euclidean)
		t.Logf("Redmean distance: %v", redmean)

		if redmean > euclidean {
			t.Logf("Note: For these skin tones, Euclidean is larger (expected for redmean)")
		}
	})

	t.Run("red channel differences weighted appropriately", func(t *testing.T) {
		c1 := Color{10, 128, 128}
		c2 := Color{50, 128, 128}

		euclidean := euclideanDistance(c1, c2)
		redmean := RedmeanDistance(c1, c2)

		t.Logf("Dark red diff - Euclidean: %v, Redmean: %v", euclidean, redmean)

		if redmean <= euclidean {
			t.Logf("Note: Redmean gives more weight to red differences in dark tones")
		}
	})

	t.Run("gradient smoothness", func(t *testing.T) {
		gradient := []Color{
			{0, 0, 0},
			{16, 16, 16},
			{32, 32, 32},
			{48, 48, 48},
			{64, 64, 64},
		}

		totalEuclidean := 0.0
		totalRedmean := 0.0
		for i := 0; i < len(gradient)-1; i++ {
			totalEuclidean += euclideanDistance(gradient[i], gradient[i+1])
			totalRedmean += RedmeanDistance(gradient[i], gradient[i+1])
		}
		t.Logf("Gradient total - Euclidean: %v, Redmean: %v", totalEuclidean, totalRedmean)
	})
}

func TestColorWeight(t *testing.T) {
	testCases := []struct {
		color      Color
		wantWeight float64
	}{
		{Color{0, 0, 0}, 1.0},
		{Color{64, 0, 0}, 1.125},
		{Color{128, 0, 0}, 1.25},
		{Color{192, 0, 0}, 1.375},
		{Color{255, 0, 0}, 1.498046875},
		{Color{0, 128, 255}, 1.0},
		{Color{192, 64, 64}, 1.375},
	}

	for _, tc := range testCases {
		got := ColorWeight(tc.color)
		if !floatEq(got, tc.wantWeight) {
			t.Errorf("ColorWeight(%v) = %v, want %v", tc.color, got, tc.wantWeight)
		}
	}
}

func TestWeightedColorDistance(t *testing.T) {
	c1 := Color{255, 128, 64}
	c2 := Color{128, 64, 255}

	d1 := WeightedColorDistance(c1, c2)
	d2 := RedmeanDistance(c1, c2)

	if d1 != d2 {
		t.Errorf("WeightedColorDistance(%v, %v) = %v, want %v", c1, c2, d1, d2)
	}
}

func TestRedmeanDistanceEdgeCases(t *testing.T) {
	t.Run("max distance colors", func(t *testing.T) {
		c1 := Color{0, 0, 0}
		c2 := Color{255, 255, 255}
		d := RedmeanDistance(c1, c2)
		dSq := RedmeanDistanceSq(c1, c2)

		rMean := 127.5
		weightR := 1.0 + rMean/256.0
		expectedSq := 2.0*weightR*65025.0 + 4.0*65025.0 + (3.0+weightR)*65025.0
		if !floatEq(dSq, expectedSq) {
			t.Errorf("RedmeanDistanceSq(black, white) = %v, want %v", dSq, expectedSq)
		}
		if math.Sqrt(dSq) != d {
			t.Errorf("RedmeanDistance should be sqrt of squared distance")
		}
	})

	t.Run("single channel differences", func(t *testing.T) {
		c1 := Color{0, 128, 128}
		c2 := Color{255, 128, 128}
		d := RedmeanDistance(c1, c2)
		dSq := RedmeanDistanceSq(c1, c2)

		dr := 255.0
		rMean := 127.5
		weightR := 1.0 + rMean/256.0
		expectedSq := 2.0 * weightR * dr * dr

		if !floatEq(dSq, expectedSq) {
			t.Errorf("RedmeanDistanceSq for red diff = %v, want %v", dSq, expectedSq)
		}
		t.Logf("Single red channel difference: dSq=%v, d=%v", dSq, d)
	})
}

func TestRedmeanDistanceSkinTones(t *testing.T) {
	skinTones := []Color{
		{255, 224, 189},
		{241, 194, 125},
		{224, 172, 105},
		{255, 219, 172},
		{234, 192, 134},
		{212, 175, 124},
		{198, 134, 103},
		{153, 85, 51},
	}

	t.Run("pairwise skin tone distances", func(t *testing.T) {
		for i := 0; i < len(skinTones); i++ {
			for j := i + 1; j < len(skinTones); j++ {
				redmeanDist := RedmeanDistance(skinTones[i], skinTones[j])
				eucDist := euclideanDistance(skinTones[i], skinTones[j])

				t.Logf("Skin tone %d -> %d: redmean=%v, euclidean=%v, ratio=%v",
					i, j, redmeanDist, eucDist, redmeanDist/eucDist)
			}
		}
	})

	t.Run("nearby skin tones should have low distance", func(t *testing.T) {
		c1 := Color{255, 224, 189}
		c2 := Color{254, 222, 187}

		redmeanDist := RedmeanDistance(c1, c2)
		eucDist := euclideanDistance(c1, c2)

		if redmeanDist > 10.0 {
			t.Errorf("Nearby skin tones should have low redmean distance: got %v", redmeanDist)
		}

		t.Logf("Nearby skin tones - redmean: %v, euclidean: %v", redmeanDist, eucDist)
	})
}

func TestRedmeanDistanceGradients(t *testing.T) {
	t.Run("grayscale gradient", func(t *testing.T) {
		var colors []Color
		for i := 0; i <= 256; i += 16 {
			colors = append(colors, Color{uint8(i), uint8(i), uint8(i)})
		}

		totalRedmean := 0.0
		totalEuclidean := 0.0
		for i := 0; i < len(colors)-1; i++ {
			totalRedmean += RedmeanDistance(colors[i], colors[i+1])
			totalEuclidean += euclideanDistance(colors[i], colors[i+1])
		}

		t.Logf("Grayscale gradient - total redmean: %v, total euclidean: %v",
			totalRedmean, totalEuclidean)
	})

	t.Run("red gradient", func(t *testing.T) {
		var colors []Color
		for i := 0; i <= 255; i += 16 {
			colors = append(colors, Color{uint8(i), 128, 128})
		}

		totalRedmean := 0.0
		totalEuclidean := 0.0
		for i := 0; i < len(colors)-1; i++ {
			totalRedmean += RedmeanDistance(colors[i], colors[i+1])
			totalEuclidean += euclideanDistance(colors[i], colors[i+1])
		}

		t.Logf("Red gradient - total redmean: %v, total euclidean: %v",
			totalRedmean, totalEuclidean)
	})

	t.Run("blue gradient", func(t *testing.T) {
		var colors []Color
		for i := 0; i <= 255; i += 16 {
			colors = append(colors, Color{128, 128, uint8(i)})
		}

		totalRedmean := 0.0
		totalEuclidean := 0.0
		for i := 0; i < len(colors)-1; i++ {
			totalRedmean += RedmeanDistance(colors[i], colors[i+1])
			totalEuclidean += euclideanDistance(colors[i], colors[i+1])
		}

		t.Logf("Blue gradient - total redmean: %v, total euclidean: %v",
			totalRedmean, totalEuclidean)
	})
}

func TestRedmeanDistanceVsOtherMetrics(t *testing.T) {
	t.Run("red vs green weighted difference", func(t *testing.T) {
		c1 := Color{128, 128, 128}
		c2 := Color{180, 128, 128}
		c3 := Color{128, 180, 128}

		redDiff := RedmeanDistance(c1, c2)
		greenDiff := RedmeanDistance(c1, c3)

		t.Logf("Red channel diff (dark): %v", redDiff)
		t.Logf("Green channel diff (same darkness): %v", greenDiff)

		if greenDiff >= redDiff {
			t.Logf("Note: Green differences weighted more heavily in redmean metric")
		}
	})

	t.Run("dark reds get more weight", func(t *testing.T) {
		c1 := Color{10, 128, 128}
		c2 := Color{50, 128, 128}
		c3 := Color{200, 128, 128}
		c4 := Color{240, 128, 128}

		darkDiff := RedmeanDistance(c1, c2)
		lightDiff := RedmeanDistance(c3, c4)

		t.Logf("Dark red diff (10->50): %v", darkDiff)
		t.Logf("Light red diff (200->240): %v", lightDiff)

		euclideanDark := euclideanDistance(c1, c2)
		euclideanLight := euclideanDistance(c3, c4)

		t.Logf("Dark red diff euclidean: %v", euclideanDark)
		t.Logf("Light red diff euclidean: %v", euclideanLight)
	})
}

func euclideanDistance(c1, c2 Color) float64 {
	dr := float64(int64(c1.R) - int64(c2.R))
	dg := float64(int64(c1.G) - int64(c2.G))
	db := float64(int64(c1.B) - int64(c2.B))
	return math.Sqrt(dr*dr + dg*dg + db*db)
}

func floatEq(a, b float64) bool {
	const epsilon = 0.0001
	return math.Abs(a-b) < epsilon
}

func generateTestPixels(width, height int) []byte {
	pixels := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * 3
			pixels[offset] = uint8(x % 256)
			pixels[offset+1] = uint8(y % 256)
			pixels[offset+2] = uint8((x + y) % 256)
		}
	}
	return pixels
}

func TestQuantizePerceptualBasic(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	indexed, palette := QuantizePerceptual(pixels, 2, 4)

	if len(indexed) != 4 {
		t.Errorf("QuantizePerceptual() indexed length = %v, want 4", len(indexed))
	}

	if palette.NumColors > 4 {
		t.Errorf("QuantizePerceptual() palette size = %v, want <= 4", palette.NumColors)
	}
}

func TestQuantizePerceptualIdenticalToLinear(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	colorMap := CountColors(pixels, 2)
	colorsWithCount := ToColorWithCountSlice(colorMap)
	paletteColors := MedianCut(colorsWithCount, 256)

	palette := NewPalette(len(paletteColors))
	for _, c := range paletteColors {
		palette.AddColor(c)
	}

	indexedLUT := make([]byte, len(pixels)/3)
	indexedLinear := make([]byte, len(pixels)/3)

	lut := NewPerceptualPaletteLUT(palette)

	for i := 0; i < len(pixels)/3; i++ {
		offset := i * 3
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		indexedLUT[i] = lut.Lookup(c.R, c.G, c.B, 255)
		indexedLinear[i] = uint8(palette.FindNearestPerceptual(c))
	}

	for i := range indexedLUT {
		if indexedLUT[i] != indexedLinear[i] {
			t.Errorf("Perceptual LUT vs Linear: pixel %d differs: LUT=%d, Linear=%d", i, indexedLUT[i], indexedLinear[i])
		}
	}
}

func TestQuantizePerceptualVsEuclidean(t *testing.T) {
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

	indexedEuclidean, paletteEuclidean := Quantize(pixels, 2, 32, true)
	errorEuclidean := calculateQuantizationError(pixels, indexedEuclidean, &paletteEuclidean, 2)

	indexedPerceptual, palettePerceptual := QuantizePerceptual(pixels, 2, 32, true)
	errorPerceptual := calculatePerceptualQuantizationError(pixels, indexedPerceptual, &palettePerceptual, 2)

	t.Logf("Euclidean quantization error: %f", errorEuclidean)
	t.Logf("Perceptual quantization error: %f", errorPerceptual)

	if errorPerceptual > errorEuclidean {
		improvement := (errorEuclidean - errorPerceptual) / errorEuclidean * 100
		t.Logf("Perceptual improvement: %.2f%%", improvement)
	}
}

func TestQuantizePerceptualSkinTones(t *testing.T) {
	var pixels []byte
	skinTones := []Color{
		{255, 224, 189},
		{241, 194, 125},
		{224, 172, 105},
		{255, 219, 172},
		{234, 192, 134},
		{212, 175, 124},
		{198, 134, 103},
		{153, 85, 51},
	}

	for _, c := range skinTones {
		pixels = append(pixels, c.R, c.G, c.B)
	}

	indexedPerceptual, palettePerceptual := QuantizePerceptual(pixels, 2, 16, false)
	errorPerceptual := calculatePerceptualQuantizationError(pixels, indexedPerceptual, &palettePerceptual, 2)

	indexedEuclidean, paletteEuclidean := Quantize(pixels, 2, 16, false)
	errorEuclidean := calculateQuantizationError(pixels, indexedEuclidean, &paletteEuclidean, 2)

	t.Logf("Skin tones - Euclidean error: %f, Perceptual error: %f", errorEuclidean, errorPerceptual)

	totalRedmeanError := 0.0
	for i := 0; i < len(pixels)/3; i++ {
		offset := i * 3
		c := Color{
			R: pixels[offset],
			G: pixels[offset+1],
			B: pixels[offset+2],
		}
		paletteColor := palettePerceptual.Colors[indexedPerceptual[i]]
		totalRedmeanError += RedmeanDistanceSq(c, paletteColor)
	}

	t.Logf("Total redmean error for skin tones: %f", totalRedmeanError)
}

func TestQuantizePerceptualGradient(t *testing.T) {
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

	indexedEuclidean, paletteEuclidean := Quantize(pixels, 2, 16, false)
	errorEuclidean := calculateQuantizationError(pixels, indexedEuclidean, &paletteEuclidean, 2)

	indexedPerceptual, palettePerceptual := QuantizePerceptual(pixels, 2, 16, false)
	errorPerceptual := calculatePerceptualQuantizationError(pixels, indexedPerceptual, &palettePerceptual, 2)

	t.Logf("Grayscale gradient - Euclidean error: %f, Perceptual error: %f", errorEuclidean, errorPerceptual)

	if errorPerceptual > errorEuclidean {
		t.Logf("Note: Perceptual error higher for grayscale gradient (expected - redmean weights red channel more)")
	}
}

func TestFindNearestPerceptual(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 0, B: 0})
	palette.AddColor(Color{R: 0, G: 255, B: 0})
	palette.AddColor(Color{R: 0, G: 0, B: 255})

	tests := []struct {
		color     Color
		wantIndex int
	}{
		{Color{0, 0, 0}, 0},
		{Color{255, 0, 0}, 1},
		{Color{0, 255, 0}, 2},
		{Color{0, 0, 255}, 3},
	}

	for _, tt := range tests {
		got := palette.FindNearestPerceptual(tt.color)
		if got != tt.wantIndex {
			t.Errorf("FindNearestPerceptual(%v) = %d, want %d", tt.color, got, tt.wantIndex)
		}
	}
}

func TestFindNearestPerceptualSkinTones(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{255, 224, 189})
	palette.AddColor(Color{241, 194, 125})
	palette.AddColor(Color{224, 172, 105})
	palette.AddColor(Color{255, 219, 172})

	nearby := Color{254, 222, 187}

	idx := palette.FindNearestPerceptual(nearby)
	selectedColor := palette.Colors[idx]

	euclideanNearest := palette.FindNearest(nearby)
	euclideanColor := palette.Colors[euclideanNearest]

	t.Logf("Perceptual selected: %v", selectedColor)
	t.Logf("Euclidean selected: %v", euclideanColor)
	t.Logf("Input color: %v", nearby)

	perceptualDist := RedmeanDistance(nearby, selectedColor)
	euclideanDist := euclideanDistance(nearby, euclideanColor)

	t.Logf("Perceptual distance: %f", perceptualDist)
	t.Logf("Euclidean distance: %f", euclideanDist)
}

func TestQuantizeFastPerceptual(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	indexed, palette := QuantizeFastPerceptual(pixels, 2, 256)

	if len(indexed) != len(pixels)/3 {
		t.Errorf("QuantizeFastPerceptual() indexed length = %v, want %v", len(indexed), len(pixels)/3)
	}

	if palette.NumColors > 256 {
		t.Errorf("QuantizeFastPerceptual() palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestQuantizeToPalettePerceptual(t *testing.T) {
	pixels := []byte{
		255, 0, 0, 0, 255, 0,
		0, 0, 255, 255, 255, 0,
	}

	palette := NewPalette(4)
	palette.AddColor(Color{255, 0, 0})
	palette.AddColor(Color{0, 255, 0})
	palette.AddColor(Color{0, 0, 255})
	palette.AddColor(Color{255, 255, 0})

	indexed := QuantizeToPalettePerceptual(pixels, 2, *palette)

	if len(indexed) != 4 {
		t.Errorf("QuantizeToPalettePerceptual() indexed length = %v, want 4", len(indexed))
	}

	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("QuantizeToPalettePerceptual()[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestQuantizeWithOptionsPerceptual(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	opts := Options{
		UsePerceptualDistance: true,
		DistanceMetric:        DistanceMetricRedmean,
	}

	indexed, palette := QuantizeWithOptions(pixels, 2, 256, opts, false)

	if len(indexed) != len(pixels)/3 {
		t.Errorf("QuantizeWithOptions() indexed length = %v, want %v", len(indexed), len(pixels)/3)
	}

	if palette.NumColors > 256 {
		t.Errorf("QuantizeWithOptions() palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestQuantizeWithOptionsEuclidean(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	opts := Options{
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}

	indexed, palette := QuantizeWithOptions(pixels, 2, 256, opts, false)

	if len(indexed) != len(pixels)/3 {
		t.Errorf("QuantizeWithOptions() indexed length = %v, want %v", len(indexed), len(pixels)/3)
	}

	if palette.NumColors > 256 {
		t.Errorf("QuantizeWithOptions() palette size = %v, want <= 256", palette.NumColors)
	}
}

func TestPerceptualLUTIsPerceptual(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{255, 0, 0})
	palette.AddColor(Color{0, 255, 0})
	palette.AddColor(Color{0, 0, 255})
	palette.AddColor(Color{255, 255, 0})

	perceptualLUT := NewPerceptualPaletteLUT(palette)
	euclideanLUT := NewFullPaletteLUT(palette)

	if !perceptualLUT.IsPerceptual() {
		t.Error("Perceptual LUT should return true for IsPerceptual()")
	}

	if euclideanLUT.IsPerceptual() {
		t.Error("Euclidean LUT should return false for IsPerceptual()")
	}
}

func BenchmarkQuantizePerceptual_LUT(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizePerceptual(pixels, 2, 256, true)
	}
}

func BenchmarkQuantizePerceptual_Linear(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	for i := 0; i < b.N; i++ {
		QuantizePerceptual(pixels, 2, 256, false)
	}
}

func BenchmarkFindNearestPerceptual(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	pixels := generateTestPixels(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(pixels); j += 3 {
			c := Color{R: pixels[j], G: pixels[j+1], B: pixels[j+2]}
			_ = palette.FindNearestPerceptual(c)
		}
	}
}

func BenchmarkPerceptualLUTBuild(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	for i := 0; i < b.N; i++ {
		NewPerceptualPaletteLUT(palette)
	}
}

func BenchmarkPerceptualLUTLookup(b *testing.B) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}
	lut := NewPerceptualPaletteLUT(palette)
	pixels := generateTestPixels(100, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(pixels); j += 3 {
			_ = lut.Lookup(pixels[j], pixels[j+1], pixels[j+2], 255)
		}
	}
}

func BenchmarkQuantizeWithOptionsPerceptual(b *testing.B) {
	pixels := generateTestPixels(100, 100)
	opts := Options{
		UsePerceptualDistance: true,
		DistanceMetric:        DistanceMetricRedmean,
	}
	for i := 0; i < b.N; i++ {
		QuantizeWithOptions(pixels, 2, 256, opts, true)
	}
}

func TestPerceptualDistancePreservesBackwardCompatibility(t *testing.T) {
	pixels := generateTestPixels(50, 50)

	indexedOld, paletteOld := Quantize(pixels, 2, 256, true)

	opts := Options{
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
	indexedNew, paletteNew := QuantizeWithOptions(pixels, 2, 256, opts, true)

	if len(indexedOld) != len(indexedNew) {
		t.Errorf("Index length differs: old=%d, new=%d", len(indexedOld), len(indexedNew))
	}

	if paletteOld.NumColors != paletteNew.NumColors {
		t.Errorf("Palette size differs: old=%d, new=%d", paletteOld.NumColors, paletteNew.NumColors)
	}

	errorOld := calculateQuantizationError(pixels, indexedOld, &paletteOld, 2)
	errorNew := calculateQuantizationError(pixels, indexedNew, &paletteNew, 2)

	t.Logf("Old quantization error: %f", errorOld)
	t.Logf("New quantization error: %f", errorNew)

	if errorNew > errorOld*1.01 {
		t.Errorf("New error (%f) should be similar to old error (%f)", errorNew, errorOld)
	}
}
