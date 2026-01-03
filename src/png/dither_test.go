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

	indexed := FloydSteinberg(pixels, *palette)

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
	indexed := FloydSteinberg([]byte{}, *palette)

	if len(indexed) != 0 {
		t.Errorf("FloydSteinberg() empty length = %v, want 0", len(indexed))
	}
}

func TestFloydSteinbergGradient(t *testing.T) {
	palette := NewPalette(2)
	palette.AddColor(Color{0, 0, 0})     // black
	palette.AddColor(Color{255, 0, 0})   // red

	// Create a gradient from black to red
	pixels := make([]byte, 6*10)
	for i := 0; i < 10; i++ {
		val := uint8(i * 25)
		pixels[i*6] = val
		pixels[i*6+1] = 0
		pixels[i*6+2] = 0
	}

	indexed := FloydSteinberg(pixels, *palette)

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

	indexed := FloydSteinberg(pixels, *palette)

	// All values should be valid bytes
	for i, idx := range indexed {
		if idx > 255 {
			t.Errorf("FloydSteinberg()[%v] = %v, want <= 255", i, idx)
		}
	}
}

func TestFloydSteinbergRow(t *testing.T) {
	palette := NewPalette(3)
	palette.AddColor(Color{0, 0, 0})
	palette.AddColor(Color{127, 127, 127})
	palette.AddColor(Color{255, 255, 255})

	pixels := []byte{128, 128, 128, 64, 64, 64}

	indexed, errors := FloydSteinbergRow(pixels, *palette, nil)

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

	indexed, _ := FloydSteinbergRow(pixels, *palette, prevErrors)

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

	result := FloydSteinberg2D(pixels, width, height, *palette)

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

	indexed := JarvisJudiceNinke(pixels, *palette)

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
	fsIdx := FloydSteinberg(pixels, *palette)

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
	fsIdx := FloydSteinberg(pixels, *palette)

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
