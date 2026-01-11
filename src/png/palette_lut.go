package png

type PaletteLUT struct {
	opaqueLUT           [64][64][64]uint8
	fullLUT             [256][256][256]uint8
	palette             *Palette
	transparentFallback bool
	useFullLUT          bool
	usePerceptual       bool
}

func NewPaletteLUT(palette *Palette) *PaletteLUT {
	lut := &PaletteLUT{
		palette:             palette,
		transparentFallback: true,
		useFullLUT:          false,
	}

	for r := 0; r < 64; r++ {
		for g := 0; g < 64; g++ {
			for b := 0; b < 64; b++ {
				fullR := uint8(r << 2)
				fullG := uint8(g << 2)
				fullB := uint8(b << 2)
				lut.opaqueLUT[r][g][b] = uint8(palette.FindNearest(Color{R: fullR, G: fullG, B: fullB}))
			}
		}
	}

	return lut
}

func NewFullPaletteLUT(palette *Palette) *PaletteLUT {
	lut := &PaletteLUT{
		palette:             palette,
		transparentFallback: true,
		useFullLUT:          true,
	}

	for r := 0; r < 256; r++ {
		for g := 0; g < 256; g++ {
			for b := 0; b < 256; b++ {
				lut.fullLUT[r][g][b] = uint8(palette.FindNearest(Color{R: uint8(r), G: uint8(g), B: uint8(b)}))
			}
		}
	}

	return lut
}

func NewPerceptualPaletteLUT(palette *Palette) *PaletteLUT {
	lut := &PaletteLUT{
		palette:             palette,
		transparentFallback: true,
		useFullLUT:          true,
		usePerceptual:       true,
	}

	for r := 0; r < 256; r++ {
		for g := 0; g < 256; g++ {
			for b := 0; b < 256; b++ {
				lut.fullLUT[r][g][b] = uint8(palette.FindNearestPerceptual(Color{R: uint8(r), G: uint8(g), B: uint8(b)}))
			}
		}
	}

	return lut
}

func (lut *PaletteLUT) Lookup(r, g, b uint8, a uint8) uint8 {
	if a != 255 && lut.transparentFallback {
		if lut.usePerceptual {
			return uint8(lut.palette.FindNearestPerceptual(Color{R: r, G: g, B: b}))
		}
		return uint8(lut.palette.FindNearest(Color{R: r, G: g, B: b}))
	}

	if lut.useFullLUT {
		return lut.fullLUT[r][g][b]
	}

	ri := r >> 2
	gi := g >> 2
	bi := b >> 2
	return lut.opaqueLUT[ri][gi][bi]
}

func (lut *PaletteLUT) SetTransparentFallback(enabled bool) {
	lut.transparentFallback = enabled
}

func (lut *PaletteLUT) TransparentFallback() bool {
	return lut.transparentFallback
}

func (lut *PaletteLUT) MemoryUsage() int {
	if lut.useFullLUT {
		return 256 * 256 * 256
	}
	return 64 * 64 * 64
}

func (lut *PaletteLUT) IsFullLUT() bool {
	return lut.useFullLUT
}

func (lut *PaletteLUT) IsPerceptual() bool {
	return lut.usePerceptual
}
