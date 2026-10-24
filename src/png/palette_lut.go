package png

import (
	"sync/atomic"
	"unsafe"
)

const paletteCacheBits = 12

// PaletteLUT memoizes nearest-color searches in a bounded direct-mapped cache.
// Collisions replace entries; a hit always compares the complete RGB key.
// Opaque lookups use an owned palette snapshot, as the old precomputed tables did.
// Transparent fallback searches the original palette, ignoring alpha in distance.
// Lookup may be called concurrently while the palette and settings are unchanged.
// Copying a PaletteLUT after its first lookup is not supported.
type PaletteLUT struct {
	cache               []atomic.Uint64
	palette             *Palette
	snapshot            Palette
	transparentFallback bool
	useFullLUT          bool
	usePerceptual       bool
}

// NewPaletteLUT preserves the legacy compact lookup's four-value RGB rounding.
// Use NewFullPaletteLUT for exact, full-precision nearest-color results.
func NewPaletteLUT(palette *Palette) *PaletteLUT {
	return newPaletteLUT(palette, false, false)
}

// NewFullPaletteLUT caches exact Euclidean searches without building a 256^3 table.
func NewFullPaletteLUT(palette *Palette) *PaletteLUT {
	return newPaletteLUT(palette, true, false)
}

// NewPerceptualPaletteLUT caches exact Redmean searches without an exhaustive table.
func NewPerceptualPaletteLUT(palette *Palette) *PaletteLUT {
	return newPaletteLUT(palette, true, true)
}

func newPaletteLUT(palette *Palette, full, perceptual bool) *PaletteLUT {
	colors := append([]Color(nil), palette.Colors[:palette.NumColors]...)
	lut := &PaletteLUT{
		palette:             palette,
		snapshot:            Palette{Colors: colors, NumColors: palette.NumColors},
		transparentFallback: true,
		useFullLUT:          full,
		usePerceptual:       perceptual,
	}
	if palette.NumColors > 1 {
		lut.cache = make([]atomic.Uint64, 1<<paletteCacheBits)
	}
	return lut
}

// Lookup returns a palette index, preserving the first index on distance ties.
// Alpha selects the legacy live-palette fallback; it is not part of RGB distance.
func (lut *PaletteLUT) Lookup(r, g, b uint8, a uint8) uint8 {
	if a != 255 && lut.transparentFallback {
		if lut.usePerceptual {
			return uint8(lut.palette.FindNearestPerceptual(Color{R: r, G: g, B: b}))
		}
		return uint8(lut.palette.FindNearest(Color{R: r, G: g, B: b}))
	}
	if len(lut.cache) == 0 {
		return 0 // Empty and single-color snapshots always choose index zero.
	}
	if !lut.useFullLUT {
		r, g, b = r&252, g&252, b&252
	}
	key := uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	slot := &lut.cache[(key*0x9e3779b1)>>(32-paletteCacheBits)]
	entry := slot.Load()
	// A separate validity bit distinguishes an unused slot from black/index zero.
	if entry>>32 != 0 && uint32(entry)&0xffffff == key {
		return uint8(entry >> 24)
	}
	c := Color{R: r, G: g, B: b}
	var index int
	if lut.usePerceptual {
		index = lut.snapshot.FindNearestPerceptual(c)
	} else {
		index = lut.snapshot.FindNearest(c)
	}
	result := uint8(index)
	slot.Store(1<<32 | uint64(result)<<24 | uint64(key))
	return result
}

// SetTransparentFallback controls whether nonopaque lookups search the live palette.
func (lut *PaletteLUT) SetTransparentFallback(enabled bool) {
	lut.transparentFallback = enabled
}

// TransparentFallback reports whether nonopaque lookups search the live palette.
func (lut *PaletteLUT) TransparentFallback() bool {
	return lut.transparentFallback
}

// MemoryUsage returns bytes of owned Go storage: the object, cache backing array,
// and palette snapshot backing array. It excludes the caller-owned palette,
// allocator size-class overhead, and process/WASM linear-memory high-water marks.
func (lut *PaletteLUT) MemoryUsage() int {
	return int(unsafe.Sizeof(*lut)) + cap(lut.cache)*int(unsafe.Sizeof(atomic.Uint64{})) +
		cap(lut.snapshot.Colors)*int(unsafe.Sizeof(Color{}))
}

// IsFullLUT reports full RGB precision, not an exhaustive table allocation.
func (lut *PaletteLUT) IsFullLUT() bool {
	return lut.useFullLUT
}

// IsPerceptual reports whether searches use Redmean distance.
func (lut *PaletteLUT) IsPerceptual() bool {
	return lut.usePerceptual
}
