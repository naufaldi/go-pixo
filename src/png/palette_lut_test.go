package png

import (
	"testing"
)

func TestNewPaletteLUT(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 255, B: 255})
	palette.AddColor(Color{R: 255, G: 0, B: 0})

	lut := NewPaletteLUT(palette)

	if lut == nil {
		t.Fatal("NewPaletteLUT returned nil")
	}

	if lut.palette != palette {
		t.Error("Palette reference not set correctly")
	}

	if !lut.transparentFallback {
		t.Error("Transparent fallback should be enabled by default")
	}
}

func TestPaletteLUT_OpaqueLookup(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 255, B: 255})
	palette.AddColor(Color{R: 255, G: 0, B: 0})

	lut := NewPaletteLUT(palette)

	tests := []struct {
		r, g, b uint8
		want    uint8
	}{
		{0, 0, 0, 0},
		{255, 255, 255, 1},
		{255, 0, 0, 2},
		{254, 254, 254, 1},
		{1, 1, 1, 0},
	}

	for _, tt := range tests {
		got := lut.Lookup(tt.r, tt.g, tt.b, 255)
		if got != tt.want {
			t.Errorf("Lookup(%d, %d, %d) = %d, want %d", tt.r, tt.g, tt.b, got, tt.want)
		}
	}
}

func TestPaletteLUT_TransparentFallback(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 255, B: 255})

	lut := NewPaletteLUT(palette)

	want := palette.FindNearest(Color{R: 128, G: 128, B: 128})
	got := lut.Lookup(128, 128, 128, 128)

	if got != uint8(want) {
		t.Errorf("Transparent fallback returned %d, want %d", got, want)
	}
}

func TestPaletteLUT_TransparentDisabled(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 0, G: 0, B: 0})
	palette.AddColor(Color{R: 255, G: 255, B: 255})

	lut := NewPaletteLUT(palette)
	lut.SetTransparentFallback(false)

	ri := 128 >> 2
	gi := 128 >> 2
	bi := 128 >> 2
	want := lut.opaqueLUT[ri][gi][bi]
	got := lut.Lookup(128, 128, 128, 128)

	if got != want {
		t.Errorf("Lookup with disabled fallback returned %d, want %d", got, want)
	}
}

func TestPaletteLUT_MatchesLinearSearch(t *testing.T) {
	palette := NewPalette(256)
	colors := []Color{
		{R: 0, G: 0, B: 0},
		{R: 63, G: 63, B: 63},
		{R: 127, G: 127, B: 127},
		{R: 191, G: 191, B: 191},
		{R: 255, G: 255, B: 255},
		{R: 255, G: 0, B: 0},
		{R: 0, G: 255, B: 0},
		{R: 0, G: 0, B: 255},
		{R: 128, G: 64, B: 32},
		{R: 32, G: 128, B: 192},
	}

	for _, c := range colors {
		palette.AddColor(c)
	}

	lut := NewPaletteLUT(palette)

	for r := 0; r < 256; r += 17 {
		for g := 0; g < 256; g += 17 {
			for b := 0; b < 256; b += 17 {
				quantR := uint8(r >> 2 << 2)
				quantG := uint8(g >> 2 << 2)
				quantB := uint8(b >> 2 << 2)

				lutIdx := lut.Lookup(uint8(r), uint8(g), uint8(b), 255)
				linearIdx := palette.FindNearest(Color{R: quantR, G: quantG, B: quantB})

				if int(lutIdx) != linearIdx {
					t.Errorf("LUT mismatch for RGB(%d,%d,%d): LUT=%d, Linear=%d (quantized to %d,%d,%d)",
						r, g, b, lutIdx, linearIdx, quantR, quantG, quantB)
				}
			}
		}
	}
}

func TestPaletteLUT_MemoryUsage(t *testing.T) {
	palette := NewPalette(256)
	for i := 0; i < 256; i++ {
		palette.AddColor(Color{R: uint8(i), G: uint8(i), B: uint8(i)})
	}

	lut := NewPaletteLUT(palette)
	memoryUsage := lut.MemoryUsage()

	expected := 64 * 64 * 64
	if memoryUsage != expected {
		t.Errorf("MemoryUsage() = %d, want %d", memoryUsage, expected)
	}

	if memoryUsage != 262144 {
		t.Errorf("Expected 262,144 entries, got %d", memoryUsage)
	}
}

func TestPaletteLUT_EmptyPalette(t *testing.T) {
	palette := NewPalette(256)
	lut := NewPaletteLUT(palette)

	result := lut.Lookup(128, 128, 128, 255)
	if result != 0 {
		t.Errorf("Empty palette lookup returned %d, want 0", result)
	}
}

func TestPaletteLUT_SingleColor(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 100, G: 150, B: 200})

	lut := NewPaletteLUT(palette)

	for r := 0; r < 256; r += 64 {
		for g := 0; g < 256; g += 64 {
			for b := 0; b < 256; b += 64 {
				result := lut.Lookup(uint8(r), uint8(g), uint8(b), 255)
				if result != 0 {
					t.Errorf("Single color palette returned %d, want 0 for RGB(%d,%d,%d)",
						result, r, g, b)
				}
			}
		}
	}
}

func TestPaletteLUT_Getters(t *testing.T) {
	palette := NewPalette(256)
	palette.AddColor(Color{R: 0, G: 0, B: 0})

	lut := NewPaletteLUT(palette)

	if lut.TransparentFallback() != true {
		t.Error("Default transparent fallback should be true")
	}

	lut.SetTransparentFallback(false)
	if lut.TransparentFallback() != false {
		t.Error("SetTransparentFallback(false) should return false")
	}
}
