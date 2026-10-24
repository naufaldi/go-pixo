package png

import (
	"bytes"
	"image/png"
	"reflect"
	"testing"
)

func TestPNGMeasuredPreprocessingProgress(t *testing.T) {
	for _, dither := range []bool{false, true} {
		name := "plain"
		if dither {
			name = "dither"
		}
		t.Run(name, func(t *testing.T) {
			const width, height = 128, 128
			pixels := make([]byte, width*height*4)
			for i := 0; i < width*height; i++ {
				pixels[i*4] = byte(i % 128 * 2)
				pixels[i*4+1] = byte(i / 128 * 2)
				pixels[i*4+2] = byte((i % 31) * 8)
				pixels[i*4+3] = 255
			}
			opts := BalancedOptions(width, height)
			opts.ColorType, opts.MaxColors, opts.Dithering = ColorRGBA, 16, dither
			enc, err := NewEncoderWithOptions(opts)
			if err != nil {
				t.Fatal(err)
			}
			want, err := enc.Encode(pixels)
			if err != nil {
				t.Fatal(err)
			}
			updates := map[string][]int{}
			opts.ProgressCallback = func(phase string, percent int) { updates[phase] = append(updates[phase], percent) }
			got, err := enc.EncodeWithOptions(pixels, opts)
			if err != nil {
				t.Fatal(err)
			}
			// Reporting must leave palette decisions, dithering and compressed pixels identical.
			if !bytes.Equal(got, want) {
				t.Fatal("reporting changed encoded bytes")
			}
			if _, err := png.Decode(bytes.NewReader(got)); err != nil {
				t.Fatal(err)
			}
			if values := updates["finalize"]; !reflect.DeepEqual(values, []int{0, 100}) {
				t.Errorf("indexed PNG skipped file writing boundaries: %v", values)
			}
			for _, phase := range []string{"histogram", "palette", "quantize", "filtering"} {
				values := updates[phase]
				if len(values) < 3 {
					t.Errorf("%s: want measured intermediate work, got %v", phase, values)
					continue
				}
				if values[0] != 0 || values[len(values)-1] != 100 {
					t.Errorf("%s boundaries: %v", phase, values)
				}
				for i, value := range values {
					if value < 0 || value > 100 || i > 0 && value <= values[i-1] {
						t.Errorf("%s: duplicate or nonmonotonic work: %v", phase, values)
						break
					}
				}
				if len(values) > 102 {
					t.Errorf("%s: unbounded callbacks: %d", phase, len(values))
				}
			}
		})
	}
}

// The reference deliberately recomputes every bucket as the original algorithm
// did. This catches changed split selection, stale child variances and tie order.
func referenceMedianCut(colors []ColorWithCount, maxColors int, quality float64) []Color {
	if len(colors) <= maxColors {
		out := make([]Color, len(colors))
		for i, c := range colors {
			out[i] = c.Color
		}
		return out
	}
	target := maxColors
	if quality < 1 {
		target = max(2, int(float64(maxColors)*(0.5+quality*0.5)))
	}
	buckets := []bucket{{colors: colors}}
	for len(buckets) < target {
		largest, variance := -1, 0.0
		for i, b := range buckets {
			v := calculateColorVariance(b.colors)
			if v > variance && len(b.colors) >= 2 {
				largest, variance = i, v
			}
		}
		if largest < 0 {
			break
		}
		left, right := splitBucketWithQuality(buckets[largest].colors, quality)
		buckets[largest].colors = left
		if len(right) > 0 {
			buckets = append(buckets, bucket{colors: right})
		}
	}
	result := make([]Color, 0, len(buckets))
	for _, b := range buckets {
		if len(b.colors) > 0 {
			result = append(result, averageColors(b.colors))
		}
	}
	return result
}

func TestMedianCutVarianceReusePreservesPalette(t *testing.T) {
	for _, quality := range []float64{0, 0.25, 0.85, 1} {
		for _, maxColors := range []int{2, 16, 256} {
			colors := make([]ColorWithCount, 1024)
			for i := range colors {
				colors[i] = ColorWithCount{Color: Color{R: byte(i), G: byte(i / 4), B: byte(i * 31)}, Count: 1 + i%11}
			}
			got := MedianCutWithQuality(colors, maxColors, quality)
			want := referenceMedianCut(colors, maxColors, quality)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("quality=%g colors=%d: changed palette selection", quality, maxColors)
			}
		}
	}
}
