package png

import (
	"bytes"
	"image"
	"image/color"
	stdpng "image/png"
	"strconv"
	"testing"
)

func TestGrayscaleReductionRejectsTransparency(t *testing.T) {
	for _, alpha := range []byte{0, 128, 254} {
		t.Run(strconv.Itoa(int(alpha)), func(t *testing.T) {
			pixels := []byte{64, 64, 64, 255, 128, 128, 128, alpha}
			if CanReduceToGrayscale(pixels, 2, 1, ColorRGBA) {
				t.Error("transparent RGBA cannot lose its alpha channel")
			}
			if _, _, err := ReduceToGrayscale(pixels, 2, 1, ColorRGBA); err != ErrCannotReduceColorType {
				t.Errorf("ReduceToGrayscale error = %v, want %v", err, ErrCannotReduceColorType)
			}
		})
	}
}

func TestEncoderLosslessReductionPreservesSamples(t *testing.T) {
	for _, tc := range []struct {
		name   string
		pixels []byte
	}{
		{"semitransparent grayscale", []byte{128, 128, 128, 128}},
		{"transparent grayscale", []byte{64, 64, 64, 0}},
		{"transparent color", []byte{12, 34, 56, 0}},
		{"opaque grayscale", []byte{128, 128, 128, 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := BalancedOptions(1, 1)
			opts.OptimizeAlpha = false // Isolate the lossless color reduction step.
			enc, err := NewEncoderWithOptions(opts)
			if err != nil {
				t.Fatal(err)
			}
			out, err := enc.Encode(tc.pixels)
			if err != nil {
				t.Fatal(err)
			}
			assertDecodedPixels(t, out, 1, 1, ColorRGBA, tc.pixels)
		})
	}
}

func TestRecompressLosslessPreservesSamples(t *testing.T) {
	for _, tc := range []struct {
		name    string
		samples color.Palette
		indexed bool
	}{
		{"grayscale alpha", color.Palette{color.NRGBA{128, 128, 128, 128}, color.NRGBA{64, 64, 64, 255}}, false},
		{"transparent grayscale", color.Palette{color.NRGBA{64, 64, 64, 0}, color.NRGBA{128, 128, 128, 128}}, false},
		{"RGBA hidden RGB", color.Palette{color.NRGBA{12, 34, 56, 0}, color.NRGBA{128, 64, 32, 128}}, false},
		{"palette grayscale alpha", color.Palette{color.NRGBA{128, 128, 128, 128}, color.NRGBA{64, 64, 64, 0}}, true},
		{"palette color alpha", color.Palette{color.NRGBA{12, 34, 56, 0}, color.NRGBA{128, 64, 32, 128}}, true},
		{"opaque colors", color.Palette{color.NRGBA{255, 0, 0, 255}, color.NRGBA{0, 255, 0, 255}, color.NRGBA{0, 0, 255, 255}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bounds := image.Rect(0, 0, 32, 8)
			var img image.Image
			if tc.indexed {
				paletted := image.NewPaletted(bounds, tc.samples)
				for i := range paletted.Pix {
					paletted.Pix[i] = byte(i % len(tc.samples))
				}
				img = paletted
			} else {
				nrgba := image.NewNRGBA(bounds)
				for y := 0; y < bounds.Dy(); y++ {
					for x := 0; x < bounds.Dx(); x++ {
						nrgba.Set(x, y, tc.samples[x%len(tc.samples)])
					}
				}
				img = nrgba
			}
			in := encodeUncompressedPNG(t, img)
			for _, template := range []string{"balanced", "quantization", "dithering"} {
				t.Run(template, func(t *testing.T) {
					opts := BalancedOptions(99, 99)
					if template != "balanced" {
						opts.MaxColors = 1
						opts.Dithering = template == "dithering"
						opts.DitheringStrength = 1
						opts.QualityTarget = 1
					}
					out, err := RecompressPNGBytesLossless(in, opts)
					if err != nil {
						t.Fatal(err)
					}
					assertPNGSampleEquality(t, in, out)
					if len(out) >= len(in) {
						t.Fatalf("fixture must exercise re-encoding, not original-byte fallback: in=%d out=%d", len(in), len(out))
					}
				})
			}
		})
	}
}

func TestRecompressLosslessPreserves16BitInput(t *testing.T) {
	bounds := image.Rect(0, 0, 32, 8)
	gray := image.NewGray16(bounds)
	rgba := image.NewRGBA64(bounds)
	nrgba := image.NewNRGBA64(bounds)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			gray.SetGray16(x, y, color.Gray16{Y: 0x1234})
			rgba.SetRGBA64(x, y, color.RGBA64{R: 0x1234, G: 0x5678, B: 0x9abc, A: 0xffff})
			alpha := uint16(0x789a)
			if x%2 == 0 {
				alpha = 0
			}
			nrgba.SetNRGBA64(x, y, color.NRGBA64{R: 0x1234, G: 0x5678, B: 0x9abc, A: alpha})
		}
	}
	for _, tc := range []struct {
		name string
		img  image.Image
	}{
		{"grayscale", gray},
		{"RGB", rgba},
		{"RGBA", nrgba},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := encodeUncompressedPNG(t, tc.img)
			if in[24] != 16 {
				t.Fatalf("fixture bit depth = %d, want 16", in[24])
			}
			opts := BalancedOptions(1, 1)
			opts.ApplyLossy(1, 1, 1)
			out, err := RecompressPNGBytesLossless(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			assertPNGSampleEquality(t, in, out)
			if !bytes.Equal(in, out) {
				t.Error("unsupported 16-bit precision must return original bytes")
			}
		})
	}
}

func TestRecompressLosslessKeepsMetadataWhenCompressionDoesNotHelp(t *testing.T) {
	pixels := bytes.Repeat([]byte{12, 34, 56, 128}, 16)
	opts := FastOptions(4, 4)
	opts.CompressionLevel = 0
	opts.FilterStrategy = FilterStrategyNone
	opts.PreserveChunks = []Chunk{
		{chunkType: ChunkSRGB, Data: []byte{0}},
		{chunkType: ChunkGAMA, Data: []byte{0, 0, 177, 143}},
	}
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	in, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	out, err := RecompressPNGBytesLossless(in, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertPNGSampleEquality(t, in, out)
	if !bytes.Equal(in, out) {
		t.Error("equally sized candidate must return original bytes")
	}
}

func encodeUncompressedPNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := stdpng.Encoder{CompressionLevel: stdpng.NoCompression}
	if err := enc.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	// Insert color metadata after the standard encoder's IHDR, before IDAT.
	in := buf.Bytes()
	gamma := Chunk{chunkType: ChunkGAMA, Data: []byte{0, 0, 177, 143}}
	return bytes.Join([][]byte{in[:33], gamma.Bytes(), in[33:]}, nil)
}

func assertPNGSampleEquality(t *testing.T, input, output []byte) {
	t.Helper()
	want, err := stdpng.Decode(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	got, err := stdpng.Decode(bytes.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != want.Bounds() {
		t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	b := want.Bounds()
	model := color.NRGBAModel
	if input[24] == 16 {
		model = color.NRGBA64Model
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			// Match source precision to avoid premultiplying hidden RGB during conversion.
			wantSample := model.Convert(want.At(x, y))
			gotSample := model.Convert(got.At(x, y))
			if gotSample != wantSample {
				t.Fatalf("pixel(%d,%d) = %#v, want %#v", x, y, gotSample, wantSample)
			}
		}
	}
	if len(output) > len(input) {
		t.Errorf("lossless output grew: in=%d out=%d", len(input), len(output))
	}
	wantChunks, err := ExtractPreserveChunks(input)
	if err != nil {
		t.Fatal(err)
	}
	gotChunks, err := ExtractPreserveChunks(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotChunks) != len(wantChunks) {
		t.Fatalf("preserved %d metadata chunks, want %d", len(gotChunks), len(wantChunks))
	}
	for i, want := range wantChunks {
		if gotChunks[i].chunkType != want.chunkType || !bytes.Equal(gotChunks[i].Data, want.Data) {
			t.Errorf("metadata chunk %d changed: got %+v, want %+v", i, gotChunks[i], want)
		}
	}
}
