package png

import (
	"testing"
)

func TestQuantizeBasic(t *testing.T) {
	// 2x2 RGB image (2*2*3 = 12 bytes)
	pixels := []byte{
		255, 0, 0,   // red
		0, 255, 0,   // green
		0, 0, 255,   // blue
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
			pixels[idx+2] = uint8((x * 2 + y) % 256)
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
	for i, idx := range indexed {
		if idx > 255 {
			t.Errorf("Quantize()[%v] = %v, want <= 255", i, idx)
		}
	}
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
