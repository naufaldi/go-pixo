package png

import (
	"testing"
)

func TestThresholdBasic(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})       // black
	palette.AddColor(Color{127, 127, 127}) // gray
	palette.AddColor(Color{255, 255, 255}) // white

	// 2 pixels: black and white
	pixels := []byte{0, 0, 0, 255, 255, 255}

	indexed := Threshold(pixels, *palette)

	if len(indexed) != 2 {
		t.Errorf("Threshold() length = %v, want 2", len(indexed))
	}

	// First pixel should map to black (idx 0)
	if indexed[0] != 0 {
		t.Errorf("Threshold()[0] = %v, want 0", indexed[0])
	}

	// Second pixel should map to white (idx 2)
	if indexed[1] != 2 {
		t.Errorf("Threshold()[1] = %v, want 2", indexed[1])
	}
}

func TestThresholdGrayPixel(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})       // black
	palette.AddColor(Color{127, 127, 127}) // gray
	palette.AddColor(Color{255, 255, 255}) // white

	// Gray pixel should map to nearest palette color (gray, idx 1)
	pixels := []byte{128, 128, 128}

	indexed := Threshold(pixels, *palette)

	if len(indexed) != 1 {
		t.Errorf("Threshold() gray length = %v, want 1", len(indexed))
	}

	if indexed[0] != 1 {
		t.Errorf("Threshold() gray = %v, want 1", indexed[0])
	}
}

func TestThresholdEmpty(t *testing.T) {
	palette := NewPalette(4)
	indexed := Threshold([]byte{}, *palette)

	if len(indexed) != 0 {
		t.Errorf("Threshold() empty length = %v, want 0", len(indexed))
	}
}

func TestFloydSteinbergBasic(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})       // black
	palette.AddColor(Color{127, 127, 127}) // gray
	palette.AddColor(Color{255, 255, 255}) // white

	// 2 pixels: black and white
	pixels := []byte{0, 0, 0, 255, 255, 255}

	indexed := FloydSteinberg(pixels, *palette, DitherMedium)

	if len(indexed) != 2 {
		t.Errorf("FloydSteinberg() length = %v, want 2", len(indexed))
	}

	// All indices should be valid
	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("FloydSteinberg()[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestFloydSteinbergEmpty(t *testing.T) {
	palette := NewPalette(4)
	indexed := FloydSteinberg([]byte{}, *palette, DitherMedium)

	if len(indexed) != 0 {
		t.Errorf("FloydSteinberg() empty length = %v, want 0", len(indexed))
	}
}

func TestFloydSteinbergGradient(t *testing.T) {
	palette := NewPalette(2)
	palette.AddColor(Color{0, 0, 0})   // black
	palette.AddColor(Color{255, 0, 0}) // red

	// Create a gradient from black to red
	pixels := make([]byte, 6*10)
	for i := 0; i < 10; i++ {
		val := uint8(i * 25)
		pixels[i*6] = val
		pixels[i*6+1] = 0
		pixels[i*6+2] = 0
	}

	indexed := FloydSteinberg(pixels, *palette, DitherMedium)

	// 60 bytes / 3 bytes per pixel = 20 pixels
	if len(indexed) != 20 {
		t.Errorf("FloydSteinberg() gradient length = %v, want 20", len(indexed))
	}

	// All indices should be valid (0 or 1)
	for i, idx := range indexed {
		if idx > 1 {
			t.Errorf("FloydSteinberg() gradient[%v] = %v, want 0 or 1", i, idx)
		}
	}
}

func TestFloydSteinbergValidByteRange(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	// Create pixels that would cause error diffusion
	pixels := make([]byte, 3*10)
	for i := range pixels {
		pixels[i] = uint8(i * 28)
	}

	indexed := FloydSteinberg(pixels, *palette, DitherMedium)

	// All values should be valid bytes
	for i, idx := range indexed {
		if int(idx) >= palette.NumColors {
			t.Errorf("FloydSteinberg()[%v] = %v, want < %d", i, idx, palette.NumColors)
		}
	}
}

func TestFloydSteinbergRow(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed, errors := FloydSteinbergRow(pixels, *palette, nil, DitherMedium)

	if len(indexed) != 2 {
		t.Errorf("FloydSteinbergRow() indexed length = %v, want 2", len(indexed))
	}

	if len(errors) != 4 {
		t.Errorf("FloydSteinbergRow() errors length = %v, want 4", len(errors))
	}
}

func TestFloydSteinbergRowWithPrevErrors(t *testing.T) {
	palette := NewPalette(2)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128}
	prevErrors := [][3]int{
		{10, 10, 10},
		{5, 5, 5},
		{0, 0, 0},
	}

	indexed, _ := FloydSteinbergRow(pixels, *palette, prevErrors, DitherMedium)

	if len(indexed) != 1 {
		t.Errorf("FloydSteinbergRow() with prev errors indexed length = %v, want 1", len(indexed))
	}
}

func TestFloydSteinberg2D(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{85, 85, 85})
	palette.AddColor(Color{170, 170, 170})
	palette.AddColor(Color{255, 255, 255})

	width, height := 4, 4
	pixels := make([]byte, width*height*3)

	// Fill with gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			val := uint8(((x + y) * 16) % 256)
			pixels[idx] = val
			pixels[idx+1] = val
			pixels[idx+2] = val
		}
	}

	result := FloydSteinberg2D(pixels, width, height, *palette, DitherMedium)

	if len(result) != width*height {
		t.Errorf("FloydSteinberg2D() result length = %v, want %v", len(result), width*height)
	}
}

func TestJarvisJudiceNinke(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := JarvisJudiceNinke(pixels, *palette, DitherMedium)

	if len(indexed) != 2 {
		t.Errorf("JarvisJudiceNinke() length = %v, want 2", len(indexed))
	}

	// All indices should be valid
	for i, idx := range indexed {
		if idx >= uint8(palette.NumColors) {
			t.Errorf("JarvisJudiceNinke()[%v] = %v, want < %v", i, idx, palette.NumColors)
		}
	}
}

func TestClampInt(t *testing.T) {
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
		result := clampInt(tt.input)
		if result != tt.expected {
			t.Errorf("clampInt(%v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestDitheringProducesValidOutput(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{85, 85, 85})
	palette.AddColor(Color{170, 170, 170})
	palette.AddColor(Color{255, 255, 255})

	// Create test pixels
	pixels := make([]byte, 3*20)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	// Test threshold
	thresholdIdx := Threshold(pixels, *palette)

	// Test Floyd-Steinberg
	fsIdx := FloydSteinberg(pixels, *palette, DitherMedium)

	// Both should produce same number of output pixels
	if len(thresholdIdx) != len(fsIdx) {
		t.Errorf("Threshold and FloydSteinberg output lengths differ: %v vs %v", len(thresholdIdx), len(fsIdx))
	}

	// Both should produce valid indices
	for i := range thresholdIdx {
		if thresholdIdx[i] >= uint8(palette.NumColors) {
			t.Errorf("Threshold[%v] = %v, want < %v", i, thresholdIdx[i], palette.NumColors)
		}
	}

	for i := range fsIdx {
		if fsIdx[i] >= uint8(palette.NumColors) {
			t.Errorf("FloydSteinberg[%v] = %v, want < %v", i, fsIdx[i], palette.NumColors)
		}
	}
}

func TestDitheringWithSmallPalette(t *testing.T) {
	palette := NewPalette(2)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{255, 255, 255})

	// Black to white gradient
	pixels := make([]byte, 3*10)
	for i := 0; i < 10; i++ {
		val := uint8(i * 25)
		pixels[i*3] = val
		pixels[i*3+1] = val
		pixels[i*3+2] = val
	}

	thresholdIdx := Threshold(pixels, *palette)
	fsIdx := FloydSteinberg(pixels, *palette, DitherMedium)

	// Both should produce 10 output pixels
	if len(thresholdIdx) != 10 || len(fsIdx) != 10 {
		t.Errorf("Dithering output length mismatch: T=%v, FS=%v, want 10", len(thresholdIdx), len(fsIdx))
	}

	// All indices should be 0 or 1
	for i := range thresholdIdx {
		if thresholdIdx[i] > 1 {
			t.Errorf("Threshold[%v] = %v, want 0 or 1", i, thresholdIdx[i])
		}
	}

	for i := range fsIdx {
		if fsIdx[i] > 1 {
			t.Errorf("FloydSteinberg[%v] = %v, want 0 or 1", i, fsIdx[i])
		}
	}
}

func TestFloydSteinbergWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	result := FloydSteinbergWithStrength(pixels, *palette, 0.5)

	if len(result) != 2 {
		t.Errorf("FloydSteinbergWithStrength() length = %v, want 2", len(result))
	}
}

func TestFloydSteinbergWithZeroStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	result := FloydSteinbergWithStrength(pixels, *palette, 0.0)

	if len(result) != 2 {
		t.Errorf("FloydSteinbergWithStrength(0) length = %v, want 2", len(result))
	}
}

func TestFloydSteinbergWithFullStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	result := FloydSteinbergWithStrength(pixels, *palette, 1.0)

	if len(result) != 2 {
		t.Errorf("FloydSteinbergWithStrength(1) length = %v, want 2", len(result))
	}
}

func TestFloydSteinbergRowWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed, errors := FloydSteinbergRowWithStrength(pixels, *palette, nil, 0.5)

	if len(indexed) != 2 {
		t.Errorf("FloydSteinbergRowWithStrength() indexed length = %v, want 2", len(indexed))
	}

	if len(errors) != 4 {
		t.Errorf("FloydSteinbergRowWithStrength() errors length = %v, want 4", len(errors))
	}
}

func TestFloydSteinberg2DWithStrength(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{85, 85, 85})
	palette.AddColor(Color{170, 170, 170})
	palette.AddColor(Color{255, 255, 255})

	width, height := 4, 4
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			val := uint8(((x + y) * 16) % 256)
			pixels[idx] = val
			pixels[idx+1] = val
			pixels[idx+2] = val
		}
	}

	result := FloydSteinberg2DWithStrength(pixels, width, height, *palette, 0.5)

	if len(result) != width*height {
		t.Errorf("FloydSteinberg2DWithStrength() result length = %v, want %v", len(result), width*height)
	}
}

func TestJarvisJudiceNinkeWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := JarvisJudiceNinkeWithStrength(pixels, *palette, 0.5)

	if len(indexed) != 2 {
		t.Errorf("JarvisJudiceNinkeWithStrength() length = %v, want 2", len(indexed))
	}
}

func TestJarvisJudiceNinkeWithZeroStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := JarvisJudiceNinkeWithStrength(pixels, *palette, 0.0)

	if len(indexed) != 2 {
		t.Errorf("JarvisJudiceNinkeWithStrength(0) length = %v, want 2", len(indexed))
	}
}

func TestSierra2Row(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := Sierra2Row(pixels, *palette, DitherMedium)

	if len(indexed) != 2 {
		t.Errorf("Sierra2Row() length = %v, want 2", len(indexed))
	}
}

func TestSierra2RowWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := Sierra2RowWithStrength(pixels, *palette, 0.5)

	if len(indexed) != 2 {
		t.Errorf("Sierra2RowWithStrength() length = %v, want 2", len(indexed))
	}
}

func TestStucki(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := Stucki(pixels, *palette, DitherMedium)

	if len(indexed) != 2 {
		t.Errorf("Stucki() length = %v, want 2", len(indexed))
	}
}

func TestStuckiWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed := StuckiWithStrength(pixels, *palette, 0.5)

	if len(indexed) != 2 {
		t.Errorf("StuckiWithStrength() length = %v, want 2", len(indexed))
	}
}

func TestDitherMethodConstants(t *testing.T) {
	if DitherMethodNone != 0 {
		t.Errorf("DitherMethodNone = %v, want 0", DitherMethodNone)
	}
	if DitherMethodFloydSteinberg != 1 {
		t.Errorf("DitherMethodFloydSteinberg = %v, want 1", DitherMethodFloydSteinberg)
	}
	if DitherMethodJarvisJudiceNinke != 2 {
		t.Errorf("DitherMethodJarvisJudiceNinke = %v, want 2", DitherMethodJarvisJudiceNinke)
	}
	if DitherMethodSierra2Row != 3 {
		t.Errorf("DitherMethodSierra2Row = %v, want 3", DitherMethodSierra2Row)
	}
	if DitherMethodStucki != 4 {
		t.Errorf("DitherMethodStucki = %v, want 4", DitherMethodStucki)
	}
}

func TestDitherStrengthConstants(t *testing.T) {
	if DitherNone != 0.0 {
		t.Errorf("DitherNone = %v, want 0.0", DitherNone)
	}
	if DitherLow != 0.25 {
		t.Errorf("DitherLow = %v, want 0.25", DitherLow)
	}
	if DitherMedium != 0.5 {
		t.Errorf("DitherMedium = %v, want 0.5", DitherMedium)
	}
	if DitherHigh != 0.75 {
		t.Errorf("DitherHigh = %v, want 0.75", DitherHigh)
	}
	if DitherMaximum != 1.0 {
		t.Errorf("DitherMaximum = %v, want 1.0", DitherMaximum)
	}
}

func TestDitherFunction(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	result := Dither(pixels, *palette, DitherMethodFloydSteinberg, DitherMedium)

	if len(result) != 2 {
		t.Errorf("Dither() FloydSteinberg length = %v, want 2", len(result))
	}

	// Test with none method (should act like threshold)
	resultNone := Dither(pixels, *palette, DitherMethodNone, DitherMedium)
	if len(resultNone) != 2 {
		t.Errorf("Dither() None length = %v, want 2", len(resultNone))
	}
}

func TestDither2D(t *testing.T) {
	palette := NewPalette(4)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{85, 85, 85})
	palette.AddColor(Color{170, 170, 170})
	palette.AddColor(Color{255, 255, 255})

	width, height := 4, 4
	pixels := make([]byte, width*height*3)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := (y*width + x) * 3
			val := uint8(((x + y) * 16) % 256)
			pixels[idx] = val
			pixels[idx+1] = val
			pixels[idx+2] = val
		}
	}

	result := Dither2D(pixels, width, height, *palette, DitherMethodFloydSteinberg, DitherMedium)

	if len(result) != width*height {
		t.Errorf("Dither2D() result length = %v, want %v", len(result), width*height)
	}
}

func TestFloydSteinbergConsistencyWithStrength(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	// Compare explicit strength with DitherStrength wrapper
	explicitResult := FloydSteinbergWithStrength(pixels, *palette, 0.5)
	wrapperResult := FloydSteinberg(pixels, *palette, DitherMedium)

	if len(explicitResult) != len(wrapperResult) {
		t.Errorf("FloydSteinberg strength consistency: lengths differ")
	}
}
