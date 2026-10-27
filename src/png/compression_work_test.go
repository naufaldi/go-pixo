package png

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"image"
	stdpng "image/png"
	"io"
	"math/rand"
	"testing"
)

func TestScratchWinnerBytesAndReconstruction(t *testing.T) {
	for _, strategy := range []FilterStrategy{FilterStrategyAdaptive, FilterStrategyEntropy, FilterStrategyCombined} {
		for _, bpp := range []int{1, 3, 4} {
			rows := [][]byte{{10, 0}, {0, 10}, {5, 5}, {255, 128}, {0, 0}}
			scratch := NewAdaptiveScratch(2*bpp, bpp)
			var previous []byte
			for y, samples := range rows {
				row := bytes.Repeat(samples, bpp)
				wantType, wantBytes := SelectFilterWithStrategy(row, previous, bpp, strategy)
				gotType, gotBytes := SelectFilterWithStrategyAndScratch(row, previous, bpp, strategy, scratch)
				if gotType != wantType || !bytes.Equal(gotBytes, wantBytes) {
					t.Errorf("strategy=%d bpp=%d row=%d: got (%d,%v), want (%d,%v)", strategy, bpp, y, gotType, gotBytes, wantType, wantBytes)
				}
				var reconstructed []byte
				switch gotType {
				case FilterNone:
					reconstructed = ReconstructNone(gotBytes)
				case FilterSub:
					reconstructed = ReconstructSub(gotBytes, bpp)
				case FilterUp:
					reconstructed = ReconstructUp(gotBytes, previous)
				case FilterAverage:
					reconstructed = ReconstructAverage(gotBytes, previous, bpp)
				case FilterPaeth:
					reconstructed = ReconstructPaeth(gotBytes, previous, bpp)
				}
				if !bytes.Equal(reconstructed, row) {
					t.Errorf("strategy=%d bpp=%d row=%d: reconstructed %v, want %v", strategy, bpp, y, reconstructed, row)
				}
				previous = row
			}
		}
	}
}

func TestPNGCompressionWorkPixelRoundTrip(t *testing.T) {
	for _, strategy := range []FilterStrategy{FilterStrategyAdaptive, FilterStrategyEntropy, FilterStrategyCombined} {
		opts := FastOptions(2, 5)
		opts.FilterStrategy = strategy
		pixels := []byte{10, 10, 10, 255, 0, 0, 0, 255, 0, 0, 0, 255, 10, 10, 10, 255,
			5, 5, 5, 255, 5, 5, 5, 255, 255, 128, 64, 128, 128, 64, 32, 0, 0, 0, 0, 255, 0, 0, 0, 255}
		enc, err := NewEncoderWithOptions(opts)
		if err != nil {
			t.Fatal(err)
		}
		out, err := enc.Encode(pixels)
		if err != nil {
			t.Fatal(err)
		}
		assertDecodedPixels(t, out, 2, 5, ColorRGBA, pixels)
		idat, err := IDATDataBytesWithOptions(pixels, 2, 5, ColorRGBA, opts)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(idat, concatIDATData(parsePNGChunks(t, out))) {
			t.Error("IDAT helper and PNG writer differ")
		}
	}
}

func TestScratchEntropyPreservesUnscaledOrdering(t *testing.T) {
	rows := [][]byte{
		{9, 11, 15, 14, 5, 12, 3, 13, 4, 10, 9, 7, 3},
		{12, 1, 8, 5, 2, 15, 0, 4, 5, 1, 7, 6, 7},
	}
	scratch := NewAdaptiveScratch(13, 1)
	defer scratch.Release()
	var previous []byte
	var expectedScanlines []byte
	for y, row := range rows {
		wantType, wantBytes := SelectFilterWithStrategy(row, previous, 1, FilterStrategyEntropy)
		expectedScanlines = append(expectedScanlines, byte(wantType))
		expectedScanlines = append(expectedScanlines, wantBytes...)
		gotType, gotBytes := SelectFilterWithStrategyAndScratch(row, previous, 1, FilterStrategyEntropy, scratch)
		if gotType != wantType || !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("row %d: scratch (%d,%v), allocating (%d,%v)", y, gotType, gotBytes, wantType, wantBytes)
		}
		previous = row
	}
	opts := FastOptions(13, 2)
	opts.ColorType = ColorGrayscale
	opts.FilterStrategy = FilterStrategyEntropy
	opts.OptimalDeflate = true
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	pixels := append(append([]byte(nil), rows[0]...), rows[1]...)
	out, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	// Entropy's last floating-point bits can differ across native and WASM.
	// Preserve the allocating selector's exact stream on the current platform.
	wantIDAT, err := buildZlibData(expectedScanlines, opts)
	if err != nil {
		t.Fatal(err)
	}
	if gotIDAT := concatIDATData(parsePNGChunks(t, out)); !bytes.Equal(gotIDAT, wantIDAT) {
		t.Errorf("scratch IDAT (%d bytes) differs from allocating selection (%d bytes)", len(gotIDAT), len(wantIDAT))
	}
	decoded, err := stdpng.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range pixels {
		r, g, b, a := decoded.At(i%13, i/13).RGBA()
		if r != uint32(want)*257 || g != r || b != r || a != 65535 {
			t.Errorf("pixel %d: got (%d,%d,%d,%d), want gray %d", i, r, g, b, a, want)
		}
	}
}

func TestStoredPNGSkipsOptimalWorkAndPreservesMetadata(t *testing.T) {
	// Exceeds 65535 scanline bytes, requiring multiple stored blocks.
	opts := FastOptions(257, 65)
	opts.CompressionLevel = 0
	opts.OptimalDeflate = true
	opts.FilterStrategy = FilterStrategyNone
	opts.PreserveChunks = []Chunk{{chunkType: ChunkGAMA, Data: []byte{0, 0, 177, 143}}}
	deflateUpdates := 0
	opts.ProgressCallback = func(phase string, progress int) {
		if phase == "deflate" && progress > 0 && progress < 100 {
			deflateUpdates++
		}
	}
	pixels := bytes.Repeat([]byte{12, 34, 56, 128}, opts.Width*opts.Height)
	enc, err := NewEncoderWithOptions(opts)
	if err != nil {
		t.Fatal(err)
	}
	out, err := enc.Encode(pixels)
	if err != nil {
		t.Fatal(err)
	}
	assertDecodedPixels(t, out, opts.Width, opts.Height, ColorRGBA, pixels)
	chunks, err := ExtractPreserveChunks(out)
	if err != nil || len(chunks) != 1 || !bytes.Equal(chunks[0].Data, opts.PreserveChunks[0].Data) {
		t.Fatalf("metadata=%v err=%v", chunks, err)
	}
	if deflateUpdates != 0 {
		t.Errorf("stored path reported %d optimization steps", deflateUpdates)
	}
	idat := concatIDATData(parsePNGChunks(t, out))
	raw := idat[2 : len(idat)-4]
	for block := 0; ; block++ {
		if len(raw) < 5 || raw[0]&6 != 0 {
			t.Fatalf("block %d is not a stored block", block)
		}
		final := raw[0]&1 != 0
		n := int(binary.LittleEndian.Uint16(raw[1:3]))
		if binary.LittleEndian.Uint16(raw[3:5]) != ^uint16(n) || len(raw) < n+5 {
			t.Fatalf("invalid stored block %d", block)
		}
		raw = raw[n+5:]
		if final {
			if block != 1 || len(raw) != 0 {
				t.Fatalf("final block=%d trailing=%d", block, len(raw))
			}
			break
		}
	}
}

func TestMinimalPNGFallbackPreservesCallMetadata(t *testing.T) {
	base := FastOptions(2, 2)
	enc, err := NewEncoderWithOptions(base)
	if err != nil {
		t.Fatal(err)
	}
	opts := base
	opts.OptimalDeflate = true
	opts.EnsureSizeNotLarger = true
	opts.OriginalFileSize = 1 // Exercise the raw-pixel encoder's fallback.
	opts.PreserveChunks = []Chunk{{chunkType: ChunkSRGB, Data: []byte{0}}}
	pixels := bytes.Repeat([]byte{12, 34, 56, 128}, 4)
	out, err := enc.EncodeWithOptions(pixels, opts)
	if err != nil {
		t.Fatal(err)
	}
	assertDecodedPixels(t, out, 2, 2, ColorRGBA, pixels)
	chunks, err := ExtractPreserveChunks(out)
	if err != nil || len(chunks) != 1 || chunks[0].chunkType != ChunkSRGB {
		t.Fatalf("fallback lost call metadata: %v err=%v", chunks, err)
	}
}

func TestLosingByteCandidateEncodedOnce(t *testing.T) {
	// A compact one-pixel grayscale PNG. The RGBA candidate is larger.
	input, err := hex.DecodeString("89504e470d0a1a0a0000000d49484452000000010000000108000000003a7e9b550000000a49444154789c63d00200002c002bcc7ae8e60000000049454e44ae426082")
	if err != nil {
		t.Fatal(err)
	}
	opts := FastOptions(1, 1)
	opts.FilterStrategy = FilterStrategyNone
	opts.EnsureSizeNotLarger = true
	opts.OptimalDeflate = true
	opts.ZopfliIterations = 20
	starts := 0
	opts.ProgressCallback = func(phase string, progress int) {
		if phase == "deflate" && progress == 0 {
			starts++
		}
	}
	out, err := RecompressPNGBytesLossless(input, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, input) || starts != 1 {
		t.Fatalf("losing candidate: original=%v encode starts=%d, want exact original and one encode", bytes.Equal(out, input), starts)
	}
}

func TestPNGCompressionWorkIterationBudget(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		iterations, override int
		want                 []byte
	}{
		{"one", 1, 0, []byte{100}},
		{"two", 2, 0, []byte{50, 100}},
		{"capped", 20, 0, []byte{33, 66, 100}},
		{"default", 0, 0, []byte{33, 66, 100}},
		{"explicit config takes precedence", 1, 2, []byte{50, 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := FastOptions(2, 2)
			opts.OptimalDeflate = true
			opts.ZopfliIterations = tc.iterations
			opts.OptimalConfig.MaxIterations = tc.override
			var updates []byte
			opts.ProgressCallback = func(phase string, progress int) {
				if phase == "deflate" && progress > 0 {
					updates = append(updates, byte(progress))
				}
			}
			_, err := IDATDataBytesWithOptions([]byte{0, 10, 5, 5}, 2, 2, ColorGrayscale, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(updates, tc.want) {
				t.Errorf("progress=%v, want %v", updates, tc.want)
			}
		})
	}
}

func TestScratchEarlyTerminationKeepsWinnerBytes(t *testing.T) {
	for _, strategy := range []FilterStrategy{FilterStrategyAdaptive, FilterStrategyEntropy} {
		scratch := NewAdaptiveScratchWithConfig(2, 1, FilterSelectionConfig{
			EarlyTerminationEnabled: true, MinFiltersToTry: 2,
			FilterSelectionThreshold: 100, EarlyTerminationThreshold: 100,
		})
		row := []byte{10, 0}
		typ, filtered := SelectFilterWithStrategyAndScratch(row, nil, 1, strategy, scratch)
		if typ != FilterNone || !bytes.Equal(filtered, row) {
			t.Errorf("strategy=%d winner=(%d,%v), want None,%v", strategy, typ, filtered, row)
		}
	}
}

func TestPNGCompressionWorkCorpus(t *testing.T) {
	for _, name := range []string{"gradient", "screenshot", "photo", "incompressible"} {
		t.Run(name, func(t *testing.T) {
			const width, height = 32, 32
			pixels := make([]byte, width*height*4)
			rng := rand.New(rand.NewSource(49))
			for i := range pixels {
				x, y := (i/4)%width, i/(width*4)
				switch name {
				case "gradient":
					pixels[i] = byte(x + y)
				case "screenshot":
					pixels[i] = byte((x/8 + y/8) % 4 * 60)
				case "photo":
					pixels[i] = byte(x + y + rng.Intn(24))
				default:
					pixels[i] = byte(rng.Intn(256))
				}
			}
			img := &image.NRGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
			input := encodeUncompressedPNG(t, img)
			opts := MaxOptions(width, height)
			out, err := RecompressPNGBytesLossless(input, opts)
			if err != nil {
				t.Fatal(err)
			}
			assertPNGSampleEquality(t, input, out)
			t.Logf("input=%d output=%d", len(input), len(out))
		})
	}
}

func BenchmarkPNGCompressionWork(b *testing.B) {
	for _, iterations := range []int{3, 20} {
		name := "3"
		if iterations == 20 {
			name = "20"
		}
		b.Run(name, func(b *testing.B) {
			pixels := generateGradientData(100, 100, 3)
			opts := FastOptions(100, 100)
			opts.OptimalDeflate = true
			opts.OptimalConfig.MaxIterations = iterations
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				data, err := IDATDataBytesWithOptions(pixels, 100, 100, ColorRGB, opts)
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(len(data)), "output-B")
			}
		})
	}
}

func TestPNGIDATScanlinesMatchAllocatingSelector(t *testing.T) {
	for _, strategy := range []FilterStrategy{FilterStrategyAdaptive, FilterStrategyEntropy, FilterStrategyCombined} {
		pixels := []byte{10, 0, 0, 10, 5, 5}
		opts := FastOptions(2, 3)
		opts.FilterStrategy = strategy
		idat, err := IDATDataBytesWithOptions(pixels, 2, 3, ColorGrayscale, opts)
		if err != nil {
			t.Fatal(err)
		}
		r, err := zlib.NewReader(bytes.NewReader(idat))
		if err != nil {
			t.Fatal(err)
		}
		got, readErr := io.ReadAll(r)
		closeErr := r.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read=%v close=%v", readErr, closeErr)
		}
		var want []byte
		var previous []byte
		for y := 0; y < 3; y++ {
			row := pixels[y*2 : y*2+2]
			typ, filtered := SelectFilterWithStrategy(row, previous, 1, strategy)
			want = append(want, byte(typ))
			want = append(want, filtered...)
			previous = row
		}
		if !bytes.Equal(got, want) {
			t.Errorf("strategy=%d scanlines=%v want=%v", strategy, got, want)
		}
	}
}
