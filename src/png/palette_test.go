package png

import (
	"testing"
)

func TestNewPalette(t *testing.T) {
	tests := []struct {
		name       string
		maxColors  int
		wantCap    int
		wantNum    int
	}{
		{"max 256", 256, 256, 0},
		{"max 128", 128, 128, 0},
		{"max 64", 64, 64, 0},
		{"max 16", 16, 16, 0},
		{"max 8", 8, 8, 0},
		{"max 1", 1, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPalette(tt.maxColors)
			if cap(p.Colors) != tt.wantCap {
				t.Errorf("NewPalette() capacity = %v, want %v", cap(p.Colors), tt.wantCap)
			}
			if p.NumColors != tt.wantNum {
				t.Errorf("NewPalette() NumColors = %v, want %v", p.NumColors, tt.wantNum)
			}
		})
	}
}

func TestPaletteAddColor(t *testing.T) {
	p := NewPalette(4)

	tests := []struct {
		name       string
		color      Color
		wantIdx    int
		wantNum    int
	}{
		{"add red", Color{255, 0, 0}, 0, 1},
		{"add green", Color{0, 255, 0}, 1, 2},
		{"add blue", Color{0, 0, 255}, 2, 3},
		{"add white", Color{255, 255, 255}, 3, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := p.AddColor(tt.color)
			if idx != tt.wantIdx {
				t.Errorf("AddColor() = %v, want %v", idx, tt.wantIdx)
			}
			if p.NumColors != tt.wantNum {
				t.Errorf("AddColor() NumColors = %v, want %v", p.NumColors, tt.wantNum)
			}
		})
	}

	// Test adding to full palette
	idx := p.AddColor(Color{128, 128, 128})
	if idx != -1 {
		t.Errorf("AddColor() to full palette = %v, want -1", idx)
	}
}

func TestPaletteFindNearest(t *testing.T) {
	p := NewPalette(4)
	p.AddColor(Color{0, 0, 0})       // black, idx 0
	p.AddColor(Color{255, 0, 0})     // red, idx 1
	p.AddColor(Color{0, 255, 0})     // green, idx 2
	p.AddColor(Color{255, 255, 255}) // white, idx 3

	tests := []struct {
		name    string
		color   Color
		wantIdx int
	}{
		{"find black", Color{0, 0, 0}, 0},
		{"find red", Color{255, 0, 0}, 1},
		{"find near red", Color{250, 10, 10}, 1},
		{"find near green", Color{10, 250, 10}, 2},
		{"find white", Color{255, 255, 255}, 3},
		{"find gray", Color{128, 128, 128}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := p.FindNearest(tt.color)
			if idx != tt.wantIdx {
				t.Errorf("FindNearest() = %v, want %v", idx, tt.wantIdx)
			}
		})
	}
}

func TestPaletteGetColor(t *testing.T) {
	p := NewPalette(3)
	p.AddColor(Color{255, 0, 0})
	p.AddColor(Color{0, 255, 0})
	p.AddColor(Color{0, 0, 255})

	tests := []struct {
		name    string
		idx     int
		want    Color
		wantErr bool
	}{
		{"get idx 0", 0, Color{255, 0, 0}, false},
		{"get idx 1", 1, Color{0, 255, 0}, false},
		{"get idx 2", 2, Color{0, 0, 255}, false},
		{"out of bounds", 5, Color{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.GetColor(tt.idx)
			if !tt.wantErr && got != tt.want {
				t.Errorf("GetColor() = %v, want %v", got, tt.want)
			}
		})
	}
}
