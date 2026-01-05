package jpeg

import (
	"testing"
)

func TestRGBToYCbCr(t *testing.T) {
	tests := []struct {
		r, g, b      uint8
		ey, ecb, ecr uint8
	}{
		{0, 0, 0, 0, 128, 128},
		{255, 255, 255, 255, 128, 128},
		{255, 0, 0, 77, 85, 255},
		{0, 255, 0, 149, 43, 21},
		{0, 0, 255, 29, 255, 107},
	}

	for _, tt := range tests {
		y, cb, cr := RGBToYCbCr(tt.r, tt.g, tt.b)
		if y != tt.ey || cb != tt.ecb || cr != tt.ecr {
			t.Errorf("RGB(%d,%d,%d): got YCbCr(%d,%d,%d), want YCbCr(%d,%d,%d)",
				tt.r, tt.g, tt.b, y, cb, cr, tt.ey, tt.ecb, tt.ecr)
		}
	}
}

func TestColorRoundTrip(t *testing.T) {
	// Sample colors to test round-trip within reasonable tolerance
	samples := []struct {
		r, g, b uint8
	}{
		{0, 0, 0},
		{255, 255, 255},
		{255, 0, 0},
		{0, 255, 0},
		{0, 0, 255},
		{128, 128, 128},
		{100, 150, 200},
	}

	for _, s := range samples {
		y, cb, cr := RGBToYCbCr(s.r, s.g, s.b)
		r, g, b := YCbCrToRGB(y, cb, cr)

		// Check with tolerance of 2 (due to integer rounding in both directions)
		if absDiff(r, s.r) > 2 || absDiff(g, s.g) > 2 || absDiff(b, s.b) > 2 {
			t.Errorf("RoundTrip RGB(%d,%d,%d): got RGB(%d,%d,%d) via YCbCr(%d,%d,%d)",
				s.r, s.g, s.b, r, g, b, y, cb, cr)
		}
	}
}

func absDiff(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
