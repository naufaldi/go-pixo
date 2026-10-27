package jpeg

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"testing"
)

// Size and decodability alone do not catch a lost DC or AC backtracking path.
func TestEncoder_TrellisFidelity(t *testing.T) {
	const width, height = 32, 32
	fixtures := []struct {
		name  string
		pixel func(x, y, channel int) byte
	}{
		{"black", func(x, y, channel int) byte { return 0 }},
		{"white", func(x, y, channel int) byte { return 255 }},
		{"black-white-blocks", func(x, y, channel int) byte { return byte(((x/8 + y/8) % 2) * 255) }},
		{"gradient", func(x, y, channel int) byte { return byte((x + y) * 255 / (width + height - 2)) }},
		{"texture", func(x, y, channel int) byte { return byte((x*37 + y*71 + x*y*13 + channel*x*29) % 256) }},
	}
	for _, fixture := range fixtures {
		for _, progressive := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/progressive=%t", fixture.name, progressive), func(t *testing.T) {
				pixels := make([]byte, width*height*3)
				for y := 0; y < height; y++ {
					for x := 0; x < width; x++ {
						for c := 0; c < 3; c++ {
							pixels[(y*width+x)*3+c] = fixture.pixel(x, y, c)
						}
					}
				}
				mse := func(trellis bool) float64 {
					opts := FastOptions(width, height, 100)
					opts.Progressive = progressive
					opts.TrellisQuant = trellis
					encoder, err := NewEncoder(opts)
					if err != nil {
						t.Fatal(err)
					}
					data, err := encoder.Encode(pixels)
					if err != nil {
						t.Fatal(err)
					}
					decoded, _, err := image.Decode(bytes.NewReader(data))
					if err != nil {
						t.Fatal(err)
					}
					var squaredError float64
					for y := 0; y < height; y++ {
						for x := 0; x < width; x++ {
							r, g, b, _ := decoded.At(x, y).RGBA()
							for c, value := range []uint32{r, g, b} {
								diff := float64(value>>8) - float64(pixels[(y*width+x)*3+c])
								squaredError += diff * diff
							}
						}
					}
					return squaredError / float64(len(pixels))
				}
				ordinary, trellis := mse(false), mse(true)
				// Allow one extra squared intensity level per channel for rate savings.
				// Constant blocks must remain exact at quality 100.
				allowance := 1.0
				if fixture.name == "black" || fixture.name == "white" || fixture.name == "black-white-blocks" {
					allowance = 0
				}
				t.Logf("decoded MSE: ordinary=%g trellis=%g", ordinary, trellis)
				if trellis > ordinary+allowance {
					t.Errorf("trellis MSE %g exceeds ordinary %g + %g", trellis, ordinary, allowance)
				}
			})
		}
	}
}

func TestTrellisPreservesExactCoefficients(t *testing.T) {
	for _, dc := range []float32{-1024, 1016} {
		t.Run(fmt.Sprint(dc), func(t *testing.T) {
			var dct, table [64]float32
			for i := range dct {
				dct[i] = float32((i*137)%1001 - 500)
				table[i] = 1
			}
			dct[0] = dc
			got := TrellisOptimizeWithConfig(dct, table, TrellisConfig{Lambda: 1e6})
			want := QuantizeBlock(dct, table)
			if got != want {
				t.Errorf("exact quantization lost coefficients:\ngot  %v\nwant %v", got, want)
			}
		})
	}
}

func TestTrellisLambdaChangesSelectedPath(t *testing.T) {
	var dct, table [64]float32
	for i := range table {
		table[i] = 1
	}
	dct[0], dct[1] = 320, 1.25
	for _, test := range []struct {
		lambda float64
		wantAC int16
	}{{0.01, 0}, {100, 1}} {
		got := TrellisOptimizeWithConfig(dct, table, TrellisConfig{Lambda: test.lambda})
		if got[0] != 320 || got[1] != test.wantAC {
			t.Errorf("lambda=%g: DC=%d AC=%d, want DC=320 AC=%d", test.lambda, got[0], got[1], test.wantAC)
		}
	}
}

func TestEncoder_TrellisSelection(t *testing.T) {
	// A one-level gradient has weak AC energy that a rate-focused path drops.
	pixels := make([]byte, 8*8*3)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := byte(128 + x/4)
			copy(pixels[(y*8+x)*3:], []byte{v, v, v})
		}
	}
	encode := func(trellis bool, lambda float64) []byte {
		opts := FastOptions(8, 8, 100)
		opts.TrellisQuant, opts.TrellisLambda = trellis, lambda
		encoder, err := NewEncoder(opts)
		if err != nil {
			t.Fatal(err)
		}
		data, err := encoder.Encode(pixels)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	ordinary := encode(false, 1)
	rateFocused, fidelityFocused := encode(true, 0.01), encode(true, 100)
	if bytes.Equal(rateFocused, fidelityFocused) || bytes.Equal(rateFocused, ordinary) {
		t.Error("selected trellis mode and lambda must change the encoded path")
	}
}

// Exhaustive suffix search checks path reconstruction and the cost of long runs,
// ZRL and EOB against the symbols the encoder actually writes.
func TestTrellisMatchesExhaustiveSuffix(t *testing.T) {
	var dct, table [64]float32
	for i := range table {
		table[i] = 1
	}
	dct[Zigzag[61]], dct[Zigzag[62]], dct[Zigzag[63]] = 2.25, -1.75, 1.25
	for _, isLuminance := range []bool{true, false} {
		for _, lambda := range []float64{1, 10} {
			cost := func(block [64]int16) float64 {
				bits := 0
				for _, run := range RunLengthEncode(ZigzagReorder(block)) {
					_, length := huffmanTables.EncodeAC(run.RunLength, run.Size, isLuminance)
					bits += int(length) + int(run.Size)
				}
				distortion := 0.0
				for i, value := range block {
					diff := float64(dct[i]) - float64(value)
					distortion += diff * diff
				}
				return float64(bits) + lambda*distortion
			}
			bestCost := math.Inf(1)
			for a := -1; a <= 5; a++ {
				for b := -5; b <= 1; b++ {
					for c := -2; c <= 4; c++ {
						var block [64]int16
						block[Zigzag[61]], block[Zigzag[62]], block[Zigzag[63]] = int16(a), int16(b), int16(c)
						bestCost = math.Min(bestCost, cost(block))
					}
				}
			}
			got := trellisOptimize(dct, table, TrellisConfig{Lambda: lambda}, isLuminance)
			if gotCost := cost(got); math.Abs(gotCost-bestCost) > 1e-9 {
				t.Errorf("luminance=%t lambda=%g reconstructed cost=%g, exhaustive minimum=%g; block=%v", isLuminance, lambda, gotCost, bestCost, got)
			}
		}
	}
}

func TestTrellisDoesNotAllocatePerBlock(t *testing.T) {
	var dct, table [64]float32
	for i := range table {
		dct[i] = float32(i*17 - 500)
		table[i] = 8
	}
	allocs := testing.AllocsPerRun(10, func() {
		_ = TrellisOptimizeFull(dct, table, 1)
	})
	if allocs != 0 {
		t.Errorf("trellis allocated %g times per block; scratch storage must be reusable", allocs)
	}
}

// The encoder must run trellis with the rate tables of the component being
// coded; equal quantization tables isolate the luma/chroma rate difference.
func TestEncoder_TrellisUsesComponentRateTables(t *testing.T) {
	opts := BalancedOptions(8, 8, 75)
	opts.TrellisQuant = true
	opts.UseSIMD = false
	encoder, err := NewEncoder(opts)
	if err != nil {
		t.Fatalf("NewEncoder() error = %v", err)
	}
	standard := NewQuantizationTables(75)
	qt := &QuantizationTables{Luminance: standard.Luminance, Chrominance: standard.Luminance}

	tests := []struct {
		name        string
		isLuminance bool
	}{
		{"luminance", true},
		{"chrominance", false},
	}

	seed := uint32(1)
	var block [64]float32
	found := false
	for attempt := 0; attempt < 256 && !found; attempt++ {
		for i := range block {
			seed = seed*1664525 + 1013904223
			block[i] = float32(seed>>24) - 128
		}
		dct := ForwardDCT(block)
		found = trellisOptimize(dct, qt.Luminance, encoder.trellisConfig, true) !=
			trellisOptimize(dct, qt.Luminance, encoder.trellisConfig, false)
	}
	if !found {
		t.Fatal("no block found where luminance and chrominance trellis differ")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := ZigzagReorder(trellisOptimize(ForwardDCT(block), qt.Luminance, encoder.trellisConfig, tt.isLuminance))
			if got := encoder.quantizeBlock(block, tt.isLuminance, qt); got != want {
				t.Fatalf("quantizeBlock(isLuminance=%t) = %v, want %v", tt.isLuminance, got, want)
			}
		})
	}
}
