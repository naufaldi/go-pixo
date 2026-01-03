package png

import (
	"testing"
)

func TestMedianCutBasic(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 10},
		{Color{255, 0, 0}, 10},
		{Color{0, 255, 0}, 10},
		{Color{0, 0, 255}, 10},
	}

	result := MedianCut(colors, 4)

	if len(result) != 4 {
		t.Errorf("MedianCut() = %v colors, want 4", len(result))
	}
}

func TestMedianCutFewerColors(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 10},
		{Color{255, 0, 0}, 10},
		{Color{0, 255, 0}, 10},
		{Color{0, 0, 255}, 10},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() = %v colors, want 2", len(result))
	}
}

func TestMedianCutMoreColorsThanInput(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 10},
		{Color{255, 0, 0}, 10},
	}

	result := MedianCut(colors, 4)

	// Should return all input colors since fewer than maxColors
	if len(result) != 2 {
		t.Errorf("MedianCut() = %v colors, want 2 (input count)", len(result))
	}
}

func TestMedianCutEmpty(t *testing.T) {
	result := MedianCut([]ColorWithCount{}, 4)

	if len(result) != 0 {
		t.Errorf("MedianCut() on empty = %v, want 0", len(result))
	}
}

func TestMedianCutSingleColor(t *testing.T) {
	colors := []ColorWithCount{
		{Color{128, 128, 128}, 100},
	}

	result := MedianCut(colors, 4)

	if len(result) != 1 {
		t.Errorf("MedianCut() on single color = %v, want 1", len(result))
	}
}

func TestMedianCutMax256(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 1},
		{Color{255, 0, 0}, 1},
	}

	result := MedianCut(colors, 256)

	// Should return all colors since fewer than 256
	if len(result) != 2 {
		t.Errorf("MedianCut() with max 256 = %v colors, want 2", len(result))
	}
}

func TestMedianCut2Colors(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 50},
		{Color{255, 255, 255}, 50},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() 2 colors = %v, want 2", len(result))
	}
}

func TestMedianCut4Colors(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 25},
		{Color{255, 0, 0}, 25},
		{Color{0, 255, 0}, 25},
		{Color{0, 0, 255}, 25},
	}

	result := MedianCut(colors, 4)

	if len(result) != 4 {
		t.Errorf("MedianCut() 4 colors = %v, want 4", len(result))
	}
}

func TestMedianCut8Colors(t *testing.T) {
	colors := make([]ColorWithCount, 8)
	for i := 0; i < 8; i++ {
		colors[i] = ColorWithCount{
			Color{uint8(i * 32), uint8(i * 32), uint8(i * 32)},
			10,
		}
	}

	result := MedianCut(colors, 8)

	if len(result) != 8 {
		t.Errorf("MedianCut() 8 colors = %v, want 8", len(result))
	}
}

func TestMedianCutRedChannel(t *testing.T) {
	// Colors that differ mainly in red channel
	colors := []ColorWithCount{
		{Color{0, 128, 128}, 10},
		{Color{64, 128, 128}, 10},
		{Color{128, 128, 128}, 10},
		{Color{192, 128, 128}, 10},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() red channel test = %v, want 2", len(result))
	}
}

func TestMedianCutGreenChannel(t *testing.T) {
	// Colors that differ mainly in green channel
	colors := []ColorWithCount{
		{Color{128, 0, 128}, 10},
		{Color{128, 64, 128}, 10},
		{Color{128, 128, 128}, 10},
		{Color{128, 192, 128}, 10},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() green channel test = %v, want 2", len(result))
	}
}

func TestMedianCutBlueChannel(t *testing.T) {
	// Colors that differ mainly in blue channel
	colors := []ColorWithCount{
		{Color{128, 128, 0}, 10},
		{Color{128, 128, 64}, 10},
		{Color{128, 128, 128}, 10},
		{Color{128, 128, 192}, 10},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() blue channel test = %v, want 2", len(result))
	}
}

func TestMedianCutWithCounts(t *testing.T) {
	// Colors with different frequencies
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 100},  // Most frequent
		{Color{255, 0, 0}, 50},
		{Color{0, 255, 0}, 25},
		{Color{0, 0, 255}, 10},
	}

	result := MedianCut(colors, 2)

	if len(result) != 2 {
		t.Errorf("MedianCut() with counts = %v, want 2", len(result))
	}
}

func TestAverageColors(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 1},
		{Color{255, 255, 255}, 1},
	}

	avg := averageColors(colors)

	// Average should be (127, 127, 127) or close to it
	if avg.R < 120 || avg.R > 135 {
		t.Errorf("averageColors() R = %v, want ~127", avg.R)
	}
	if avg.G < 120 || avg.G > 135 {
		t.Errorf("averageColors() G = %v, want ~127", avg.G)
	}
	if avg.B < 120 || avg.B > 135 {
		t.Errorf("averageColors() B = %v, want ~127", avg.B)
	}
}

func TestAverageColorsWeighted(t *testing.T) {
	colors := []ColorWithCount{
		{Color{0, 0, 0}, 3},        // Weight 3
		{Color{255, 255, 255}, 1},  // Weight 1
	}

	avg := averageColors(colors)

	// Average should be (63, 63, 63) = (0*3 + 255*1) / 4
	if avg.R < 60 || avg.R > 66 {
		t.Errorf("averageColors() weighted R = %v, want ~63", avg.R)
	}
}

func TestAverageColorsSingle(t *testing.T) {
	colors := []ColorWithCount{
		{Color{100, 150, 200}, 5},
	}

	avg := averageColors(colors)

	if avg.R != 100 || avg.G != 150 || avg.B != 200 {
		t.Errorf("averageColors() single = %v, want (100, 150, 200)", avg)
	}
}
