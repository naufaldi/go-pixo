package wasm

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"image/color"
	stdpng "image/png"
	"io"
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
	defer r.Close()
	result, err := io.ReadAll(r)
	if err != nil {
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
	var outputs [][]byte
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
