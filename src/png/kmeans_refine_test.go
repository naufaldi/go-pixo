package png

import (
	"testing"
)

func TestToColorCount(t *testing.T) {
	colorMap := map[Color]int{
		{R: 255, G: 0, B: 0}:     100,
		{R: 0, G: 255, B: 0}:     50,
		{R: 0, G: 0, B: 255}:     75,
		{R: 128, G: 128, B: 128}: 25,
	}

	result := ToColorCount(colorMap)

	if len(result) != 4 {
		t.Errorf("expected 4 colors, got %d", len(result))
	}

	if result[0].Count != 100 || result[0].Color != (Color{R: 255, G: 0, B: 0}) {
		t.Errorf("expected first color to be red with count 100, got count %d", result[0].Count)
	}

	if result[1].Count != 75 || result[1].Color != (Color{R: 0, G: 0, B: 255}) {
		t.Errorf("expected second color to be blue with count 75, got count %d", result[1].Count)
	}

	if result[2].Count != 50 || result[2].Color != (Color{R: 0, G: 255, B: 0}) {
		t.Errorf("expected third color to be green with count 50, got count %d", result[2].Count)
	}

	if result[3].Count != 25 || result[3].Color != (Color{R: 128, G: 128, B: 128}) {
		t.Errorf("expected fourth color to be gray with count 25, got count %d", result[3].Count)
	}
}

func TestToColorCountEmpty(t *testing.T) {
	result := ToColorCount(map[Color]int{})
	if len(result) != 0 {
		t.Errorf("expected 0 colors, got %d", len(result))
	}
}

func TestRefinePaletteKmeansEmptyColors(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 2,
	}
	palette.Colors[0] = Color{R: 255, G: 0, B: 0}
	palette.Colors[1] = Color{R: 0, G: 255, B: 0}

	result := RefinePaletteKmeans(palette, []ColorCount{}, 3)

	if len(result) != 2 {
		t.Errorf("expected 2 colors, got %d", len(result))
	}
	if result[0] != (Color{R: 255, G: 0, B: 0}) {
		t.Errorf("expected first color to remain red")
	}
	if result[1] != (Color{R: 0, G: 255, B: 0}) {
		t.Errorf("expected second color to remain green")
	}
}

func TestRefinePaletteKmeansEmptyPalette(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 0,
	}

	colors := []ColorCount{
		{Color: Color{R: 255, G: 0, B: 0}, Count: 10},
		{Color: Color{R: 0, G: 255, B: 0}, Count: 5},
	}

	result := RefinePaletteKmeans(palette, colors, 3)

	if len(result) != 0 {
		t.Errorf("expected 0 colors, got %d", len(result))
	}
}

func TestRefinePaletteKmeansSingleColor(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 2,
	}
	palette.Colors[0] = Color{R: 128, G: 128, B: 128}
	palette.Colors[1] = Color{R: 64, G: 64, B: 64}

	colors := []ColorCount{
		{Color: Color{R: 255, G: 0, B: 0}, Count: 100},
	}

	result := RefinePaletteKmeans(palette, colors, 3)

	if len(result) != 2 {
		t.Errorf("expected 2 colors, got %d", len(result))
	}
}

func TestRefinePaletteKmeansPreservesPaletteSize(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 8),
		NumColors: 4,
	}
	palette.Colors[0] = Color{R: 255, G: 0, B: 0}
	palette.Colors[1] = Color{R: 0, G: 255, B: 0}
	palette.Colors[2] = Color{R: 0, G: 0, B: 255}
	palette.Colors[3] = Color{R: 255, G: 255, B: 0}

	colors := []ColorCount{
		{Color: Color{R: 250, G: 10, B: 10}, Count: 50},
		{Color: Color{R: 10, G: 240, B: 10}, Count: 40},
		{Color: Color{R: 10, G: 10, B: 240}, Count: 30},
		{Color: Color{R: 240, G: 240, B: 10}, Count: 20},
		{Color: Color{R: 128, G: 128, B: 128}, Count: 10},
	}

	result := RefinePaletteKmeans(palette, colors, 3)

	if len(result) != 4 {
		t.Errorf("expected palette size to be preserved at 4, got %d", len(result))
	}
}

func TestRefinePaletteKmeansConvergence(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 2,
	}
	palette.Colors[0] = Color{R: 100, G: 100, B: 100}
	palette.Colors[1] = Color{R: 200, G: 200, B: 200}

	colors := []ColorCount{
		{Color: Color{R: 50, G: 50, B: 50}, Count: 100},
		{Color: Color{R: 150, G: 150, B: 150}, Count: 100},
		{Color: Color{R: 250, G: 250, B: 250}, Count: 100},
	}

	result3 := RefinePaletteKmeans(palette, colors, 3)
	result10 := RefinePaletteKmeans(palette, colors, 10)

	if result3[0] != result10[0] || result3[1] != result10[1] {
		t.Errorf("algorithm did not converge - 3 and 10 iterations should yield same result")
	}
}

func TestRefinePaletteKmeansImprovesQuality(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 2,
	}
	palette.Colors[0] = Color{R: 0, G: 0, B: 0}
	palette.Colors[1] = Color{R: 255, G: 255, B: 255}

	colors := []ColorCount{
		{Color: Color{R: 100, G: 100, B: 100}, Count: 100, R: 100, G: 100, B: 100},
		{Color: Color{R: 150, G: 150, B: 150}, Count: 200, R: 150, G: 150, B: 150},
		{Color: Color{R: 200, G: 200, B: 200}, Count: 100, R: 200, G: 200, B: 200},
	}

	initialPalette := []Color{
		{R: 0, G: 0, B: 0},
		{R: 255, G: 255, B: 255},
	}
	initialError := calculateTotalDistance(colors, initialPalette)
	refined := RefinePaletteKmeans(palette, colors, 3)
	refinedError := calculateTotalDistance(colors, refined)

	if refinedError >= initialError {
		t.Errorf("expected refinement to improve or maintain quality: initial=%f, refined=%f", initialError, refinedError)
	}
}

func TestRefinePaletteKmeansMultipleIterations(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 2,
	}
	palette.Colors[0] = Color{R: 50, G: 50, B: 50}
	palette.Colors[1] = Color{R: 200, G: 200, B: 200}

	colors := []ColorCount{
		{Color: Color{R: 60, G: 60, B: 60}, Count: 100},
		{Color: Color{R: 80, G: 80, B: 80}, Count: 100},
		{Color: Color{R: 180, G: 180, B: 180}, Count: 100},
		{Color: Color{R: 220, G: 220, B: 220}, Count: 100},
	}

	error1 := calculateTotalDistance(colors, RefinePaletteKmeans(palette, colors, 1))
	error2 := calculateTotalDistance(colors, RefinePaletteKmeans(palette, colors, 2))
	error3 := calculateTotalDistance(colors, RefinePaletteKmeans(palette, colors, 3))

	if error3 > error2 {
		t.Errorf("3 iterations should not have higher error than 2: 2=%f, 3=%f", error2, error3)
	}
	if error2 > error1 {
		t.Errorf("2 iterations should not have higher error than 1: 1=%f, 2=%f", error1, error2)
	}
}

func TestRefinePaletteKmeansFullPalette(t *testing.T) {
	numColors := 256
	palette := &Palette{
		Colors:    make([]Color, numColors),
		NumColors: numColors,
	}
	for i := 0; i < numColors; i++ {
		palette.Colors[i] = Color{R: uint8(i), G: uint8(i), B: uint8(i)}
	}

	colors := make([]ColorCount, numColors)
	for i := 0; i < numColors; i++ {
		r := float64(i)
		colors[i] = ColorCount{
			Color: Color{R: uint8(i), G: uint8(i), B: uint8(i)},
			Count: 1,
			R:     r,
			G:     r,
			B:     r,
		}
	}

	result := RefinePaletteKmeans(palette, colors, 3)

	if len(result) != numColors {
		t.Errorf("expected %d colors, got %d", numColors, len(result))
	}

	for i := 0; i < numColors; i++ {
		dist := colorDistance(result[i], colors[i].Color)
		if dist > 1 {
			t.Errorf("color %d changed significantly: got %v, expected %v, dist=%f", i, result[i], colors[i].Color, dist)
		}
	}
}

func TestRefinePaletteKmeansTwoIterationsNoRegression(t *testing.T) {
	palette := &Palette{
		Colors:    make([]Color, 4),
		NumColors: 3,
	}
	palette.Colors[0] = Color{R: 100, G: 100, B: 100}
	palette.Colors[1] = Color{R: 150, G: 150, B: 150}
	palette.Colors[2] = Color{R: 200, G: 200, B: 200}

	colors := []ColorCount{
		{Color: Color{R: 100, G: 100, B: 100}, Count: 50},
		{Color: Color{R: 150, G: 150, B: 150}, Count: 50},
		{Color: Color{R: 200, G: 200, B: 200}, Count: 50},
	}

	error1 := calculateTotalDistance(colors, RefinePaletteKmeans(palette, colors, 1))
	error2 := calculateTotalDistance(colors, RefinePaletteKmeans(palette, colors, 2))

	if error2 > error1 {
		t.Errorf("additional iteration should not increase error: 1 iter=%f, 2 iter=%f", error1, error2)
	}
}

func calculateTotalDistance(colors []ColorCount, palette []Color) float64 {
	var total float64
	for _, c := range colors {
		bestDist := colorDistance(c.Color, palette[0])
		for i := 1; i < len(palette); i++ {
			dist := colorDistance(c.Color, palette[i])
			if dist < bestDist {
				bestDist = dist
			}
		}
		total += bestDist * float64(c.Count)
	}
	return total
}

func TestColorCountStruct(t *testing.T) {
	cc := ColorCount{
		Color: Color{R: 100, G: 150, B: 200},
		Count: 42,
		R:     100.5,
		G:     150.5,
		B:     200.5,
	}

	if cc.Color.R != 100 {
		t.Errorf("expected Color.R to be 100, got %d", cc.Color.R)
	}
	if cc.Color.G != 150 {
		t.Errorf("expected Color.G to be 150, got %d", cc.Color.G)
	}
	if cc.Color.B != 200 {
		t.Errorf("expected Color.B to be 200, got %d", cc.Color.B)
	}
	if cc.Count != 42 {
		t.Errorf("expected Count to be 42, got %d", cc.Count)
	}
	if cc.R != 100.5 {
		t.Errorf("expected R to be 100.5, got %f", cc.R)
	}
	if cc.G != 150.5 {
		t.Errorf("expected G to be 150.5, got %f", cc.G)
	}
	if cc.B != 200.5 {
		t.Errorf("expected B to be 200.5, got %f", cc.B)
	}
}
