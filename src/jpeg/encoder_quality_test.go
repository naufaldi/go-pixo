package jpeg

import (
	"bytes"
	"image"
	"image/draw"
	stdjpeg "image/jpeg"
	"math"
	"testing"
)

// psnr decodes an encoded JPEG and compares it with the source RGB pixels.
func psnr(t *testing.T, encoded []byte, source *image.RGBA) float64 {
	t.Helper()
	decoded, err := stdjpeg.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := source.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, decoded, b.Min, draw.Src)
	var squared float64
	for i := range rgba.Pix {
		if i%4 == 3 {
			continue
		}
		d := float64(rgba.Pix[i]) - float64(source.Pix[i])
		squared += d * d
	}
	mse := squared / float64(b.Dx()*b.Dy()*3)
	if mse == 0 {
		return math.Inf(1)
	}
	return 10 * math.Log10(255*255/mse)
}

// TestEncoderOptionsKeepQuality guards against entropy-coding options that
// decode to the wrong image. Progressive and optimized Huffman output used to
// decode at ~10-20 dB; they must now match the baseline's quality exactly.
func TestEncoderOptionsKeepQuality(t *testing.T) {
	img := photoLike(96, 72, 11)
	rgb := make([]byte, 96*72*3)
	for i := 0; i < 96*72; i++ {
		copy(rgb[i*3:i*3+3], img.Pix[i*4:i*4+3])
	}
	encode := func(t *testing.T, opts Options) []byte {
		t.Helper()
		encoder, err := NewEncoder(opts)
		if err != nil {
			t.Fatal(err)
		}
		out, err := encoder.Encode(rgb)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, subsampling := range []Subsampling{Subsampling420, Subsampling444} {
		base := BalancedOptions(96, 72, 85)
		base.Subsampling = subsampling
		base.Progressive, base.OptimizeHuffman, base.TrellisQuant = false, false, false
		baseline := encode(t, base)
		baselinePSNR := psnr(t, baseline, img)
		if baselinePSNR < 30 {
			t.Fatalf("baseline PSNR %.2f dB is implausibly low", baselinePSNR)
		}
		tests := []struct {
			name                           string
			progressive, optimize, trellis bool
		}{
			{"progressive", true, false, false},
			{"optimized Huffman", false, true, false},
			{"progressive and optimized Huffman", true, true, false},
			{"trellis", false, false, true},
			{"trellis with progressive", true, true, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				opts := base
				opts.Progressive, opts.OptimizeHuffman, opts.TrellisQuant = tt.progressive, tt.optimize, tt.trellis
				out := encode(t, opts)
				got := psnr(t, out, img)
				if tt.trellis {
					// Trellis trades a little distortion for size.
					if got < baselinePSNR-1.5 {
						t.Fatalf("PSNR %.2f dB, baseline %.2f dB", got, baselinePSNR)
					}
				} else if got != baselinePSNR {
					t.Fatalf("PSNR %.2f dB differs from baseline %.2f dB; entropy coding must not change pixels", got, baselinePSNR)
				}
				if !tt.trellis && len(out) >= len(baseline) {
					t.Fatalf("size %d not smaller than baseline %d", len(out), len(baseline))
				}
			})
		}
	}
}

func TestEncoderGrayscaleProgressive(t *testing.T) {
	gray := grayLike(40, 24, 5)
	opts := BalancedOptions(40, 24, 80)
	opts.ColorType = ColorGrayscale
	opts.Progressive, opts.TrellisQuant = true, false
	encoder, err := NewEncoder(opts)
	if err != nil {
		t.Fatal(err)
	}
	out, err := encoder.Encode(gray.Pix)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := stdjpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	var squared float64
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			d := float64(decoded.(*image.Gray).GrayAt(x, y).Y) - float64(gray.GrayAt(x, y).Y)
			squared += d * d
		}
	}
	if got := 10 * math.Log10(255*255/(squared/(40*24))); got < 30 {
		t.Fatalf("grayscale progressive PSNR %.2f dB", got)
	}
}
