package png

import "math"

// Color represents an RGB color.
type Color struct {
	R, G, B uint8
}

// ColorWithCount extends Color with frequency information.
type ColorWithCount struct {
	Color
	Count int
}

// Palette represents an indexed color palette.
type Palette struct {
	Colors    []Color
	NumColors int
}

// NewPalette creates a new palette with the specified maximum number of colors.
func NewPalette(maxColors int) *Palette {
	return &Palette{
		Colors:    make([]Color, maxColors),
		NumColors: 0,
	}
}

// AddColor adds a color to the palette and returns its index.
// If the palette is full, it returns -1.
func (p *Palette) AddColor(c Color) int {
	if p.NumColors >= len(p.Colors) {
		return -1
	}
	p.Colors[p.NumColors] = c
	p.NumColors++
	return p.NumColors - 1
}

// FindNearest finds the index of the nearest color in the palette to the given color.
// Uses Euclidean distance in RGB space.
func (p *Palette) FindNearest(c Color) int {
	if p.NumColors == 0 {
		return 0
	}

	bestIdx := 0
	bestDist := uint64(math.MaxUint64)

	for i := 0; i < p.NumColors; i++ {
		dr := int64(c.R) - int64(p.Colors[i].R)
		dg := int64(c.G) - int64(p.Colors[i].G)
		db := int64(c.B) - int64(p.Colors[i].B)

		dist := uint64(dr*dr + dg*dg + db*db)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}

// FindNearestWithAlpha finds the nearest color considering alpha if palette has it.
func (p *Palette) FindNearestWithAlpha(c Color, alpha uint8) int {
	if p.NumColors == 0 {
		return 0
	}

	bestIdx := 0
	bestDist := uint64(math.MaxUint64)

	for i := 0; i < p.NumColors; i++ {
		paletteAlpha := p.Colors[i].R

		if alpha != paletteAlpha {
			continue
		}

		dr := int64(c.R) - int64(p.Colors[i].G)
		dg := int64(c.G) - int64(p.Colors[i].B)
		db := int64(c.B) - int64(p.Colors[i].R)

		dist := uint64(dr*dr + dg*dg + db*db)
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	return bestIdx
}

// HasAlpha returns true if the palette has colors with alpha information.
func (p *Palette) HasAlpha() bool {
	return false
}

// GetColor returns the color at the specified index.
func (p *Palette) GetColor(idx int) Color {
	if idx >= 0 && idx < p.NumColors {
		return p.Colors[idx]
	}
	return Color{}
}
