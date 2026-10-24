package png

import (
	"fmt"
	"testing"
)

// BenchmarkPaletteMemory includes construction and mapping, not just warmed hits.
// B/op is cumulative allocation per operation, not peak process or WASM memory.
func BenchmarkPaletteMemory(b *testing.B) {
	b.Run("FindNearestIndex", func(b *testing.B) {
		palette := memoryBenchmarkPalette()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			palette.FindNearestIndex(Color{31, 85, 127})
		}
	})
	for _, perceptual := range []bool{false, true} {
		for _, cached := range []bool{false, true} {
			b.Run(fmt.Sprintf("100x100/perceptual=%t/cache=%t", perceptual, cached), func(b *testing.B) {
				pixels := generateTestPixels(100, 100)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if perceptual {
						QuantizePerceptual(pixels, 2, 256, cached)
					} else {
						Quantize(pixels, 2, 256, cached)
					}
				}
			})
		}
	}
	for _, count := range []int{1_000_000, 12_000_000} {
		for _, cached := range []bool{false, true} {
			b.Run(fmt.Sprintf("Dither/%d/cache=%t", count, cached), func(b *testing.B) {
				pixels := repeatedMemoryPixels(count)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					QuantizeWithDithering(pixels, 2, 8, cached)
				}
			})
		}
	}
	b.Run("ToPalette/1000000/unique", func(b *testing.B) {
		palette := memoryBenchmarkPalette()
		pixels := make([]byte, 3_000_000)
		for i := 0; i < len(pixels)/3; i++ {
			pixels[i*3], pixels[i*3+1], pixels[i*3+2] = byte(i>>16), byte(i>>8), byte(i)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			QuantizeToPalette(pixels, 2, *palette)
		}
	})
}

func memoryBenchmarkPalette() *Palette {
	palette := NewPalette(32)
	for i := 0; i < 32; i++ {
		palette.AddColor(Color{byte(i * 7), byte(i * 3), byte(255 - i*5)})
	}
	return palette
}

func repeatedMemoryPixels(count int) []byte {
	pixels := make([]byte, count*3)
	for i := 0; i < count; i++ {
		pixels[i*3], pixels[i*3+1], pixels[i*3+2] = byte(i%32*7), byte(i%32*3), byte(255-i%32*5)
	}
	return pixels
}
