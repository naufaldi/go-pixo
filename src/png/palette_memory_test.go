package png

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"unsafe"
)

func TestPaletteLookupBoundedAllocation(t *testing.T) {
	for name, newLUT := range map[string]func(*Palette) *PaletteLUT{
		"compact": NewPaletteLUT, "exact": NewFullPaletteLUT, "perceptual": NewPerceptualPaletteLUT,
	} {
		t.Run(name, func(t *testing.T) {
			palette := memoryBenchmarkPalette()
			var lut *PaletteLUT
			allocated := paletteAllocatedBytes(func() { lut = newLUT(palette) })
			if allocated > 128*1024 {
				t.Errorf("construction allocated %d bytes; want bounded storage below 128 KiB", allocated)
			}
			before := lut.MemoryUsage()
			if before < int(unsafe.Sizeof(*lut)) || before > 128*1024 {
				t.Errorf("MemoryUsage() = %d, must include object and owned storage below 128 KiB", before)
			}
			if uint64(before+4096) < allocated {
				t.Errorf("MemoryUsage() = %d underreports %d allocated bytes beyond allocator rounding", before, allocated)
			}
			allocated = paletteAllocatedBytes(func() {
				for i := 0; i < 100_000; i++ {
					lut.Lookup(byte(i>>16), byte(i>>8), byte(i), 255)
				}
			})
			if allocated > 1024 || lut.MemoryUsage() != before {
				t.Errorf("distinct lookups grew memory: allocated=%d, reported=%d -> %d", allocated, before, lut.MemoryUsage())
			}
		})
	}
}

func TestFindNearestIndexDoesNotAllocate(t *testing.T) {
	palette := memoryBenchmarkPalette()
	allocated := paletteAllocatedBytes(func() {
		if got := palette.FindNearestIndex(Color{31, 85, 127}); got != palette.FindNearest(Color{31, 85, 127}) {
			t.Errorf("FindNearestIndex returned %d, want exact linear result", got)
		}
	})
	if allocated > 1024 {
		t.Errorf("one query allocated %d bytes; must not build a table or cache", allocated)
	}
}

func TestQuantizeSmallMappingAllocation(t *testing.T) {
	palette := memoryBenchmarkPalette()
	for name, quantize := range map[string]func([]byte, int, Palette, ...bool) []byte{
		"euclidean": QuantizeToPalette, "perceptual": QuantizeToPalettePerceptual,
	} {
		t.Run(name, func(t *testing.T) {
			allocated := paletteAllocatedBytes(func() {
				indexed := quantize([]byte{31, 85, 127}, 2, *palette)
				runtime.KeepAlive(indexed)
			})
			if allocated > 1024 {
				t.Errorf("one-pixel mapping allocated %d bytes; want linear search", allocated)
			}
		})
	}
}

func TestExactPaletteLookupParity(t *testing.T) {
	for name, newLUT := range map[string]func(*Palette) *PaletteLUT{
		"euclidean": NewFullPaletteLUT, "perceptual": NewPerceptualPaletteLUT,
	} {
		t.Run(name, func(t *testing.T) {
			palette := memoryBenchmarkPalette()
			lut := newLUT(palette)
			for i := 0; i < 100_000; i++ {
				// More distinct inputs than the cache can hold, including black.
				c := Color{byte(i * 73 >> 16), byte(i * 73 >> 8), byte(i * 73)}
				want := palette.FindNearest(c)
				if name == "perceptual" {
					want = palette.FindNearestPerceptual(c)
				}
				for _, alpha := range []byte{0, 1, 127, 254, 255} {
					if got := lut.Lookup(c.R, c.G, c.B, alpha); got != byte(want) {
						t.Fatalf("color=%v alpha=%d: got %d, want %d", c, alpha, got, want)
					}
				}
			}
		})
	}
}

func TestExactPaletteLookupTieAndBlack(t *testing.T) {
	palette := &Palette{Colors: []Color{{0, 0, 0}, {0, 4, 0}, {0, 0, 0}}, NumColors: 3}
	for _, newLUT := range []func(*Palette) *PaletteLUT{NewFullPaletteLUT, NewPerceptualPaletteLUT} {
		lut := newLUT(palette)
		for _, c := range []Color{{0, 0, 0}, {0, 2, 0}, {0, 3, 0}} {
			want := byte(0)
			if c.G == 3 {
				want = 1 // Rounding to the compact cube would incorrectly choose 0.
			}
			for repeat := 0; repeat < 2; repeat++ {
				if got := lut.Lookup(c.R, c.G, c.B, 255); got != want {
					t.Errorf("color=%v got %d, want %d", c, got, want)
				}
			}
		}
		grays := NewPalette(256)
		for i := 0; i < 256; i++ {
			grays.AddColor(Color{byte(255 - i), byte(255 - i), byte(255 - i)})
		}
		lut = newLUT(grays)
		for repeat := 0; repeat < 2; repeat++ {
			if got := lut.Lookup(0, 0, 0, 255); got != 255 {
				t.Errorf("black at final palette index: got %d, want 255", got)
			}
		}
	}
}

func TestQuantizationInputLimits(t *testing.T) {
	quantizers := map[string]func([]byte, int, int) ([]byte, Palette){
		"euclidean":  func(p []byte, c, n int) ([]byte, Palette) { return Quantize(p, c, n) },
		"perceptual": func(p []byte, c, n int) ([]byte, Palette) { return QuantizePerceptual(p, c, n) },
		"alpha":      func(p []byte, c, n int) ([]byte, Palette) { return QuantizeWithAlpha(p, c, n) },
		"dither":     func(p []byte, c, n int) ([]byte, Palette) { return QuantizeWithDithering(p, c, n) },
		"kmeans":     func(p []byte, c, n int) ([]byte, Palette) { return QuantizeWithKmeans(p, c, n, 0) },
		"options": func(p []byte, c, n int) ([]byte, Palette) {
			return QuantizeWithOptions(p, c, n, Options{})
		},
	}
	for name, quantize := range quantizers {
		// These helpers support RGB/RGBA, not grayscale, indexed, or gray-alpha.
		// Freeze the existing panic boundary rather than expanding format support.
		for _, colorType := range []int{0, 1, 3, 4, 5, 7, 255} {
			t.Run(fmt.Sprintf("%s/unsupported=%d", name, colorType), func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Error("non-RGB/RGBA input must retain its unsupported-input panic")
					}
				}()
				quantize([]byte{0, 0, 0}, colorType, 4)
			})
		}
		for _, maxColors := range []int{-1, 0, 1, 300} {
			t.Run(fmt.Sprintf("%s/maxColors=%d", name, maxColors), func(t *testing.T) {
				pixels := generateTestPixels(17, 17)
				indexed, palette := quantize(pixels, 2, maxColors)
				limit := maxColors
				if limit <= 0 || limit > 256 {
					limit = 256
				}
				if palette.NumColors < 1 || palette.NumColors > limit || len(indexed) != 289 {
					t.Fatalf("palette=%d output=%d, want 1..%d colors and 289 pixels", palette.NumColors, len(indexed), limit)
				}
				for _, index := range indexed {
					if int(index) >= palette.NumColors {
						t.Fatalf("index=%d exceeds palette size %d", index, palette.NumColors)
					}
				}
			})
		}
	}
}

func TestPaletteLookupSnapshotAndAlphaFallback(t *testing.T) {
	for name, newLUT := range map[string]func(*Palette) *PaletteLUT{
		"compact": NewPaletteLUT, "exact": NewFullPaletteLUT, "perceptual": NewPerceptualPaletteLUT,
	} {
		t.Run(name, func(t *testing.T) {
			palette := NewPalette(3)
			palette.AddColor(Color{0, 0, 0})
			palette.AddColor(Color{255, 255, 255})
			lut := newLUT(palette)
			lut.Lookup(255, 255, 255, 255) // Warm only white, leaving black uncached.
			palette.Colors[0] = Color{255, 255, 255}
			palette.AddColor(Color{127, 127, 127})
			if got := lut.Lookup(0, 0, 0, 255); got != 0 {
				t.Errorf("opaque lookup must use constructor snapshot: got %d, want 0", got)
			}
			if got := lut.Lookup(0, 0, 0, 0); got != 2 {
				t.Errorf("transparent fallback must search current palette: got %d, want 2", got)
			}
			lut.SetTransparentFallback(false)
			if got := lut.Lookup(0, 0, 0, 0); got != 0 {
				t.Errorf("disabled fallback must use snapshot: got %d, want 0", got)
			}
		})
	}
}

func TestPaletteLookupConcurrent(t *testing.T) {
	palette := memoryBenchmarkPalette()
	lut := NewFullPaletteLUT(palette)
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Go(func() {
			for i := 0; i < 2000; i++ {
				c := Color{byte(i), byte(i >> 8), byte(i * 17)}
				if got, want := lut.Lookup(c.R, c.G, c.B, 255), palette.FindNearest(c); got != byte(want) {
					t.Errorf("concurrent lookup %v: got %d, want %d", c, got, want)
					return
				}
			}
		})
	}
	workers.Wait()
}

func TestDitherAllocationScalesWithIndexedOutput(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(fmt.Sprintf("cache=%t", cached), func(t *testing.T) {
			measure := func(count int) uint64 {
				pixels := repeatedMemoryPixels(count)
				return paletteAllocatedBytes(func() {
					indexed, _ := QuantizeWithDithering(pixels, 2, 8, cached)
					runtime.KeepAlive(indexed)
				})
			}
			small, large := measure(1024), measure(100_000)
			// Fixed 32-color input isolates scratch from the color-count map.
			// Allow output rounding/GC noise, but not either full-image int buffer.
			if growth := int64(large) - int64(small); growth > 2*(100_000-1024)+16*1024 {
				t.Errorf("allocation grew by %d bytes for 98,976 output bytes", growth)
			}
		})
	}
}

func TestQuantizationExactMapping(t *testing.T) {
	quantizers := map[string]func([]byte, int) ([]byte, Palette){
		"euclidean":  func(p []byte, c int) ([]byte, Palette) { return Quantize(p, c, 4) },
		"perceptual": func(p []byte, c int) ([]byte, Palette) { return QuantizePerceptual(p, c, 4) },
		"options": func(p []byte, c int) ([]byte, Palette) {
			return QuantizeWithOptions(p, c, 4, Options{UsePerceptualDistance: true})
		},
		"kmeans": func(p []byte, c int) ([]byte, Palette) { return QuantizeWithKmeans(p, c, 4, 2) },
		"alpha":  func(p []byte, c int) ([]byte, Palette) { return QuantizeWithAlpha(p, c, 4) },
	}
	for name, quantize := range quantizers {
		for _, colorType := range []int{2, 6} {
			for _, count := range []int{1, 37, 2048} {
				t.Run(fmt.Sprintf("%s/type=%d/pixels=%d", name, colorType, count), func(t *testing.T) {
					pixels := memoryEdgePixels(count, colorType)
					original := bytes.Clone(pixels)
					indexed, palette := quantize(pixels, colorType)
					bpp := BytesPerPixel(ColorType(colorType))
					for i, got := range indexed {
						c := Color{pixels[i*bpp], pixels[i*bpp+1], pixels[i*bpp+2]}
						if name == "alpha" {
							// Preserve the existing compact constructor's RGB rounding.
							c = Color{c.R & 252, c.G & 252, c.B & 252}
						}
						want := palette.FindNearest(c)
						if name == "perceptual" || name == "options" {
							want = palette.FindNearestPerceptual(c)
						}
						if got != byte(want) {
							t.Fatalf("pixel %d: got %d, want %d", i, got, want)
						}
					}
					if !bytes.Equal(pixels, original) {
						t.Fatal("quantization modified caller's pixel storage")
					}
				})
			}
		}
	}
}

func TestDitherLegacyIndexParity(t *testing.T) {
	for _, colorType := range []int{2, 6} {
		for _, count := range []int{0, 1, 37, 2048} {
			for _, cached := range []bool{false, true} {
				t.Run(fmt.Sprintf("type=%d/pixels=%d/cache=%t", colorType, count, cached), func(t *testing.T) {
					pixels := memoryEdgePixels(count, colorType)
					original := bytes.Clone(pixels)
					indexed, palette := QuantizeWithDithering(pixels, colorType, 4, cached)
					want := legacyDitherIndices(pixels, colorType, palette)
					if !bytes.Equal(indexed, want) {
						t.Fatalf("dither indices differ from legacy one-dimensional output")
					}
					if !bytes.Equal(pixels, original) {
						t.Fatal("dither modified caller's pixel storage")
					}
				})
			}
		}
	}
}

// This deliberately retains the old image-sized error buffer as a test oracle.
// It freezes positive-only clamping and traversal across conceptual row boundaries.
func legacyDitherIndices(pixels []byte, colorType int, palette Palette) []byte {
	bpp := BytesPerPixel(ColorType(colorType))
	indexed := make([]byte, len(pixels)/bpp)
	errors := make([][3]int, len(indexed)+2)
	for i := range indexed {
		channels := [3]int{}
		for j := range channels {
			v := int(pixels[i*bpp+j]) + errors[i][j]
			channels[j] = min(255, max(0, v))
		}
		idx := palette.FindNearest(Color{byte(channels[0]), byte(channels[1]), byte(channels[2])})
		indexed[i] = byte(idx)
		c := palette.Colors[idx]
		for j, component := range [3]byte{c.R, c.G, c.B} {
			if i+1 < len(indexed) {
				errors[i+1][j] += (channels[j] - int(component)) * 7 / 16
			}
			errors[i+1][j] = min(255, max(0, errors[i+1][j]))
		}
	}
	return indexed
}

func memoryEdgePixels(count, colorType int) []byte {
	bpp := BytesPerPixel(ColorType(colorType))
	pixels := make([]byte, count*bpp)
	colors := []Color{{0, 0, 0}, {127, 127, 127}, {128, 128, 128}, {255, 255, 255}, {3, 0, 0}, {4, 0, 0}, {255, 1, 128}}
	for i := 0; i < count; i++ {
		c := colors[i%len(colors)]
		pixels[i*bpp], pixels[i*bpp+1], pixels[i*bpp+2] = c.R, c.G, c.B
		if bpp == 4 {
			pixels[i*bpp+3] = []byte{0, 1, 127, 254, 255}[i%5]
		}
	}
	return pixels
}

func paletteAllocatedBytes(run func()) uint64 {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	run()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}
