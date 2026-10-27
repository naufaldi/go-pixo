package wasm

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image"
	"image/color"
	stdpng "image/png"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/mac/go-pixo/src/png"
)

func pngScanlines(t *testing.T, data []byte) []byte {
	t.Helper()
	var idat []byte
	for offset := 8; offset+12 <= len(data); {
		n := int(binary.BigEndian.Uint32(data[offset:]))
		if string(data[offset+4:offset+8]) == "IDAT" {
			idat = append(idat, data[offset+8:offset+8+n]...)
		}
		offset += 12 + n
	}
	r, err := zlib.NewReader(bytes.NewReader(idat))
	if err != nil {
		t.Fatal(err)
	}
	result, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return result
}

// Catches dropping filterEarlyTerm before the IDAT selector. Both streams must decode identically.
func TestEncodePngAdvancedEarlyTerminationChangesSelection(t *testing.T) {
	pixels := make([]byte, 256)
	pixels[0], pixels[128] = 1, 1
	for _, strategy := range []png.FilterStrategy{png.FilterStrategyMinSum, png.FilterStrategyEntropy} {
		for _, tc := range []struct {
			early bool
			want  png.FilterType
		}{{false, png.FilterUp}, {true, png.FilterNone}} {
			out, err := EncodePngAdvanced(pixels, 128, 2, 0, 1, false, 0, false, 0, 100, 0, int(strategy), false, 0, false, tc.early, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := pngScanlines(t, out)[129]; got != byte(tc.want) {
				t.Errorf("early=%v: second row filter=%d, want %d", tc.early, got, tc.want)
			}
			img, err := stdpng.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			for i, want := range pixels {
				if got := color.GrayModel.Convert(img.At(i%128, i/128)).(color.Gray).Y; got != want {
					t.Fatalf("pixel %d=%d, want %d", i, got, want)
				}
			}
		}
	}
}

func TestPngPublicLosslessPreservesHiddenRGB(t *testing.T) {
	pixels := []byte{12, 34, 56, 0, 90, 100, 110, 128}
	for _, advanced := range []bool{false, true} {
		var out []byte
		var err error
		if advanced {
			out, err = EncodePngAdvanced(pixels, 2, 1, 6, 1, false, 2, true, 1, 0, 0, 0, true, 1, false, false, nil)
		} else {
			out, err = EncodePng(pixels, 2, 1, 6, 1, false, 2)
		}
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := stdpng.Decode(bytes.NewReader(out))
		if err != nil {
			t.Fatal(err)
		}
		for x := 0; x < 2; x++ {
			got := color.NRGBAModel.Convert(decoded.At(x, 0)).(color.NRGBA)
			want := color.NRGBA{pixels[x*4], pixels[x*4+1], pixels[x*4+2], pixels[x*4+3]}
			if got != want {
				t.Errorf("advanced=%v pixel %d=%v, want %v", advanced, x, got, want)
			}
		}
	}
}

// ApplyLossy must respect the explicit dithering switch instead of always enabling it.
func TestEncodePngAdvancedRespectsDitheringSwitch(t *testing.T) {
	pixels := make([]byte, 64*3)
	for i := 0; i < 64; i++ {
		pixels[i*3], pixels[i*3+1], pixels[i*3+2] = byte(i*4), byte(i*4), byte(i*4)
	}
	outputs := make([][]byte, 0, 3)
	for _, tc := range []struct {
		enabled  bool
		strength float64
	}{{false, 1}, {true, 0}, {true, 1}} {
		out, err := EncodePngAdvanced(pixels, 64, 1, 2, 1, true, 2, tc.enabled, tc.strength, 100, 0, 0, false, 0, false, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, out)
	}
	if !bytes.Equal(outputs[0], outputs[1]) {
		t.Error("disabled dithering differs from zero strength")
	}
	if bytes.Equal(outputs[0], outputs[2]) {
		t.Error("enabled dithering did not change output")
	}
}

// Catches treating the basic API's full default budget as a required RGB palette,
// or clearing hidden RGB when taking its RGBA fallback.
func TestEncodePngDefaultPreservesRGBA(t *testing.T) {
	for _, alpha := range []byte{128, 0, 255} {
		t.Run(strconv.Itoa(int(alpha)), func(t *testing.T) {
			pixels := []byte{12, 34, 56, alpha, 90, 100, 110, 255, 12, 34, 56, alpha, 90, 100, 110, 255}
			original := bytes.Clone(pixels)
			out, err := EncodePng(pixels, 2, 2, 6, 1, true, 256)
			if err != nil {
				t.Fatal(err)
			}
			assertPngRGBA(t, out, 2, 2, pixels)
			if alpha == 255 && out[25] != byte(png.ColorIndexed) {
				t.Fatal("opaque default bypassed palette quantization")
			}
			if !bytes.Equal(pixels, original) {
				t.Fatal("encoding modified caller pixels")
			}
		})
	}
}

func assertPngRGBA(t *testing.T, out []byte, width, height int, pixels []byte) {
	t.Helper()
	decoded, err := stdpng.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, width, height) {
		t.Fatalf("bounds=%v, want %dx%d", decoded.Bounds(), width, height)
	}
	for i := 0; i < width*height; i++ {
		got := color.NRGBAModel.Convert(decoded.At(i%width, i/width)).(color.NRGBA)
		want := color.NRGBA{pixels[i*4], pixels[i*4+1], pixels[i*4+2], pixels[i*4+3]}
		if got != want {
			t.Fatalf("pixel %d=%v, want %v", i, got, want)
		}
	}
}

// Uses the exact raw UI state and the actual derived Balanced worker settings.
// Automatic fallback must preserve samples while leaving opaque quantization active.
func TestEncodePngAdvancedPresetFallbackPreservesRGBA(t *testing.T) {
	for _, alpha := range []byte{128, 0, 255} {
		for _, tc := range []struct {
			name                string
			dithering           bool
			strength            float64
			quality, iterations int
		}{
			{"initial state", false, 0.5, 75, 0},
			{"derived Balanced worker", true, 0.4, 85, 5},
		} {
			t.Run(tc.name+"/"+strconv.Itoa(int(alpha)), func(t *testing.T) {
				pixels := []byte{12, 34, 56, alpha, 90, 100, 110, 255, 12, 34, 56, alpha, 90, 100, 110, 255}
				original := bytes.Clone(pixels)
				progress := 0
				out, err := EncodePngAdvanced(pixels, 2, 2, 6, 1, true, 256, tc.dithering, tc.strength, tc.quality, tc.iterations, 0, false, 0, false, false, func(string, int) { progress++ }, true)
				if err != nil {
					t.Fatal(err)
				}
				assertPngRGBA(t, out, 2, 2, pixels)
				if alpha == 255 && out[25] != byte(png.ColorIndexed) {
					t.Fatal("opaque preset bypassed palette quantization")
				}
				if progress == 0 || !bytes.Equal(pixels, original) {
					t.Fatal("progress was lost or caller pixels changed")
				}
			})
		}
	}
}

// Automatic RGBA fallback must retain encoder controls, independent of preset.
func TestPngRGBAFallbackRetainsFilterSelection(t *testing.T) {
	pixels := bytes.Repeat([]byte{12, 34, 56, 0, 90, 100, 110, 128}, 64)
	for _, preset := range []int{0, 1, 2, 4} {
		for _, filter := range []png.FilterStrategy{png.FilterStrategyNone, png.FilterStrategySub} {
			out, err := EncodePngAdvanced(pixels, 2, 64, 6, preset, true, 256, false, 0, 100, 0, int(filter), false, 0, false, false, nil, true)
			if err != nil {
				t.Fatal(err)
			}
			assertPngRGBA(t, out, 2, 64, pixels)
			if got := pngScanlines(t, out)[0]; got != byte(filter) {
				t.Fatalf("preset=%d filter=%d: got row filter %d", preset, filter, got)
			}
		}
	}
}

// Explicit palettes/quantizer controls stay strict, including a full palette cap.
func TestPngPublicExplicitPaletteRejectsTransparency(t *testing.T) {
	pixels := []byte{12, 34, 56, 0, 90, 100, 110, 128}
	for _, tc := range []struct {
		name   string
		encode func() ([]byte, error)
	}{
		{"basic reduced palette", func() ([]byte, error) { return EncodePng(pixels, 2, 1, 6, 1, true, 128) }},
		{"basic required full palette", func() ([]byte, error) { return EncodePng(pixels, 2, 1, 6, 1, true, 256, false) }},
		{"advanced explicit quantizer", func() ([]byte, error) {
			return EncodePngAdvanced(pixels, 2, 1, 6, 1, true, 256, true, 1, 0, 0, 0, true, 1, false, false, nil)
		}},
		{"advanced automatic reduced palette", func() ([]byte, error) {
			return EncodePngAdvanced(pixels, 2, 1, 6, 1, true, 128, false, 0, 100, 0, 0, false, 0, false, false, nil, true)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.encode()
			if err == nil || !strings.Contains(err.Error(), "transparency") || len(out) != 0 {
				t.Fatalf("explicit palette returned %d bytes, err=%v", len(out), err)
			}
		})
	}
}
