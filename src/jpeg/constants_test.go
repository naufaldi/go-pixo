package jpeg

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      uint16
		expected uint16
	}{
		{"SOI", SOI, 0xFFD8},
		{"EOI", EOI, 0xFFD9},
		{"APP0", APP0, 0xFFE0},
		{"DQT", DQT, 0xFFDB},
		{"SOF0", SOF0, 0xFFC0},
		{"SOF2", SOF2, 0xFFC2},
		{"DHT", DHT, 0xFFC4},
		{"SOS", SOS, 0xFFDA},
		{"DRI", DRI, 0xFFDD},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s: got 0x%X, want 0x%X", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestColorType(t *testing.T) {
	if ColorGrayscale != 1 {
		t.Errorf("ColorGrayscale: got %d, want 1", ColorGrayscale)
	}
	if ColorRGB != 3 {
		t.Errorf("ColorRGB: got %d, want 3", ColorRGB)
	}
}

func TestSubsampling(t *testing.T) {
	if Subsampling444 != 0 {
		t.Errorf("Subsampling444: got %d, want 0", Subsampling444)
	}
	if Subsampling420 != 1 {
		t.Errorf("Subsampling420: got %d, want 1", Subsampling420)
	}
}
