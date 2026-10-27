package png

import (
	"bytes"
	"image"
	"image/color"
	stdpng "image/png"
	"strconv"
	"testing"
)

func encodePaletteFixture(t *testing.T, pixels []byte, opts Options) *image.Paletted {
	t.Helper()
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	out, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := stdpng.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	paletted, ok := decoded.(*image.Paletted)
	if !ok || paletted.Bounds() != image.Rect(0, 0, opts.Width, opts.Height) {
		t.Fatalf("expected %dx%d indexed PNG, got %T", opts.Width, opts.Height, decoded)
	}
	return paletted
}

// Catches the encoder bypassing the selected distance metric, including while dithering.
func TestEncoderQuantizationUsesDistanceMetric(t *testing.T) {
	pixels := []byte{94, 81, 56, 4, 39, 39, 203, 217, 153}
	for _, dither := range []bool{false, true} {
		for _, tc := range []struct {
			name       string
			perceptual bool
			metric     DistanceMetric
			want       uint8
		}{
			{"euclidean", false, DistanceMetricEuclidean, 1},
			{"perceptual flag", true, DistanceMetricEuclidean, 0},
			{"redmean metric", false, DistanceMetricRedmean, 0},
		} {
			t.Run(tc.name+"/dither="+strconv.FormatBool(dither), func(t *testing.T) {
				opts := FastOptions(3, 1)
				opts.ColorType, opts.MaxColors = ColorRGB, 2
				opts.Dithering, opts.DitheringStrength = dither, 1
				opts.UsePerceptualDistance, opts.DistanceMetric = tc.perceptual, tc.metric
				decoded := encodePaletteFixture(t, pixels, opts)
				// Median cut yields (4,39,39) and (148,149,104). For the first
				// pixel Euclidean squared distances are 10153 and 9844; Redmean
				// weights green more strongly and chooses the first entry.
				wantPalette := []color.RGBA{{4, 39, 39, 255}, {148, 149, 104, 255}}
				for i, want := range wantPalette {
					if got := color.RGBAModel.Convert(decoded.Palette[i]); got != want {
						t.Fatalf("palette[%d]=%v, want %v", i, got, want)
					}
				}
				if got := decoded.ColorIndexAt(0, 0); got != tc.want {
					t.Fatalf("first palette index=%d, want %d", got, tc.want)
				}
			})
		}
	}
}

// Catches ignoring QualityTarget or substituting a file-size-driven quality reduction.
func TestEncoderQuantizationQualityChangesPaletteBudget(t *testing.T) {
	pixels := make([]byte, 64*3)
	for i := 0; i < 64; i++ {
		pixels[i*3], pixels[i*3+1], pixels[i*3+2] = byte(i*4), byte(i*4), byte(i*4)
	}
	for _, tc := range []struct{ quality, colors int }{{100, 8}, {0, 4}} {
		opts := FastOptions(64, 1)
		opts.ColorType, opts.MaxColors, opts.QualityTarget = ColorRGB, 8, tc.quality
		decoded := encodePaletteFixture(t, pixels, opts)
		if len(decoded.Palette) != tc.colors {
			t.Errorf("quality %d: palette size=%d, want %d", tc.quality, len(decoded.Palette), tc.colors)
		}
		for x := 0; x < 64; x++ {
			r, g, b, a := decoded.At(x, 0).RGBA()
			if r != g || g != b || a != 65535 {
				t.Fatalf("quality %d changed gray/opacity at %d", tc.quality, x)
			}
		}
	}
}

// Catches ignoring DitheringStrength or applying diffusion when dithering is disabled.
func TestEncoderQuantizationDitherStrengthChangesPixels(t *testing.T) {
	pixels := make([]byte, 64*3)
	for i := 0; i < 64; i++ {
		pixels[i*3], pixels[i*3+1], pixels[i*3+2] = byte(i*4), byte(i*4), byte(i*4)
	}
	original := bytes.Clone(pixels)
	opts := FastOptions(64, 1)
	opts.ColorType, opts.MaxColors, opts.Dithering = ColorRGB, 2, true
	opts.DitheringStrength = 0
	zero := encodePaletteFixture(t, pixels, opts)
	opts.DitheringStrength = 1
	full := encodePaletteFixture(t, pixels, opts)
	opts.Dithering = false
	disabled := encodePaletteFixture(t, pixels, opts)
	if bytes.Equal(zero.Pix, full.Pix) {
		t.Error("strength 0 and 1 produced identical palette indices")
	}
	if !bytes.Equal(zero.Pix, disabled.Pix) {
		t.Error("strength 0 differs from disabled dithering")
	}
	if !bytes.Equal(pixels, original) {
		t.Error("quantization modified caller pixels")
	}
	for _, decoded := range []*image.Paletted{zero, full, disabled} {
		if len(decoded.Palette) != 2 {
			t.Fatal("dithering changed palette budget")
		}
		for _, idx := range decoded.Pix {
			if idx >= 2 {
				t.Fatalf("invalid palette index %d", idx)
			}
		}
		for x := 0; x < 64; x++ {
			r, g, b, a := decoded.At(x, 0).RGBA()
			delta := int(r/257) - int(pixels[x*3])
			if r != g || g != b || a != 65535 || delta < -100 || delta > 100 {
				t.Fatalf("dithered gray ramp exceeds fixture error bound at %d: %v", x, decoded.At(x, 0))
			}
		}
	}
}

func TestQuantizationQualityRespectsOneColorLimit(t *testing.T) {
	opts := FastOptions(3, 1)
	opts.ColorType, opts.MaxColors, opts.QualityTarget = ColorRGB, 1, 0
	decoded := encodePaletteFixture(t, []byte{0, 0, 0, 100, 100, 100, 200, 200, 200}, opts)
	if len(decoded.Palette) != 1 {
		t.Fatalf("one-color limit produced %d colors", len(decoded.Palette))
	}
}

func TestEarlyTerminationStoredFallback(t *testing.T) {
	opts := FastOptions(2, 1)
	opts.FilterEarlyTermination = true
	opts.EnsureSizeNotLarger, opts.OriginalFileSize = true, 1
	pixels := []byte{12, 34, 56, 0, 10, 20, 30, 128}
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	out, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	assertDecodedPixels(t, out, 2, 1, ColorRGBA, pixels)
}

func TestEarlyTerminationRejectsUnsupportedStrategy(t *testing.T) {
	opts := FastOptions(1, 1)
	opts.FilterStrategy, opts.FilterEarlyTermination = FilterStrategyCombined, true
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Encode([]byte{1, 2, 3, 255}); err == nil {
		t.Fatal("early termination was silently ignored")
	}
}

func TestEncoderRejectsUnsupportedPaletteInputs(t *testing.T) {
	for _, tc := range []struct {
		name      string
		colorType ColorType
		pixels    []byte
	}{
		{"grayscale", ColorGrayscale, []byte{42}},
		{"translucent RGBA", ColorRGBA, []byte{12, 34, 56, 128}},
		{"hidden RGB", ColorRGBA, []byte{12, 34, 56, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := FastOptions(1, 1)
			opts.ColorType, opts.MaxColors = tc.colorType, 2
			enc, err := NewEncoderWithOptions(opts)
			if err != nil {
				t.Fatal(err)
			}
			if out, err := enc.Encode(tc.pixels); err == nil || len(out) != 0 {
				t.Fatalf("unsupported palette input returned %d bytes, err=%v", len(out), err)
			}
		})
	}
}

// Equal raw candidate lengths must not force None. A ramp becomes repeated 1s under Sub.
func TestBruteForceScoresCompressedCost(t *testing.T) {
	row := make([]byte, 256)
	for i := range row {
		row[i] = byte(i)
	}
	for _, scratch := range []bool{false, true} {
		var filter FilterType
		var filtered []byte
		if scratch {
			filter, filtered = SelectFilterWithStrategyAndScratch(row, nil, 1, FilterStrategyBruteForce, NewAdaptiveScratch(len(row), 1))
		} else {
			filter, filtered = SelectFilterWithStrategy(row, nil, 1, FilterStrategyBruteForce)
		}
		if filter != FilterSub {
			t.Fatalf("scratch=%v: filter=%d, want Sub", scratch, filter)
		}
		if !bytes.Equal(ReconstructSub(filtered, 1), row) {
			t.Fatal("selected filter bytes do not reconstruct")
		}
	}
}
