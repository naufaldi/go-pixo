package jpeg

import (
	"math"
	"math/rand/v2"
	"testing"
)

// trellisCost is the objective trellisOptimize minimizes for the AC part of a
// block: standard-table bits for every AC symbol (ZRLs, EOB) plus weighted
// distortion of every AC coefficient.
func trellisCost(dct, quant [64]float32, config TrellisConfig, isLuminance bool, out [64]int16) float64 {
	rates := &acRates[0]
	if !isLuminance {
		rates = &acRates[1]
	}
	cost, run, last := 0.0, 0, 0
	for i := 1; i < 64; i++ {
		index := Zigzag[i]
		cost += config.Lambda * calculateDistortion(dct[index], float32(out[index])*quant[index], index, config.UsePerceptual)
		if out[index] == 0 {
			run++
			continue
		}
		cost += float64((run>>4)*rates.byRun[0][15] + rates.byRun[getACCategory(out[index])][run&15])
		run, last = 0, i
	}
	if last < 63 {
		cost += float64(rates.byRun[0][0])
	}
	return cost
}

// trellisWideSearch is a plain dynamic program over every pending run length
// with seven candidates around the rounded value (and zero): slow, but a wide
// search to measure the fast candidate rule against.
func trellisWideSearch(dct, quant [64]float32, config TrellisConfig, isLuminance bool) [64]int16 {
	rates := &acRates[0]
	if !isLuminance {
		rates = &acRates[1]
	}
	type state struct {
		cost  float64
		value int16
		from  int // Run length before this coefficient, or -1 for a zero.
	}
	var trail [64][64]state // trail[i][run]: best way to reach position i with a pending run.
	for i := range trail {
		for r := range trail[i] {
			trail[i][r].cost = math.Inf(1)
		}
	}
	trail[0][0].cost = 0
	for i := 1; i < 64; i++ {
		index := Zigzag[i]
		zero := config.Lambda * calculateDistortion(dct[index], 0, index, config.UsePerceptual)
		for r := 0; r < i; r++ {
			if c := trail[i-1][r].cost + zero; c < trail[i][r+1].cost {
				trail[i][r+1] = state{cost: c, from: -1}
			}
		}
		center := int(math.Round(float64(dct[index] / quant[index])))
		for delta := -3; delta <= 3; delta++ {
			v := center + delta
			if v == 0 || v < -1023 || v > 1023 {
				continue
			}
			d := config.Lambda * calculateDistortion(dct[index], float32(v)*quant[index], index, config.UsePerceptual)
			for r := 0; r < i; r++ {
				c := trail[i-1][r].cost + float64((r>>4)*rates.byRun[0][15]+rates.byRun[getACCategory(int16(v))][r&15]) + d
				if c < trail[i][0].cost {
					trail[i][0] = state{cost: c, value: int16(v), from: r}
				}
			}
		}
	}
	best, run := trail[63][0].cost, 0
	for r := 1; r < 64; r++ {
		if c := trail[63][r].cost + float64(rates.byRun[0][0]); c < best {
			best, run = c, r
		}
	}
	var out [64]int16
	for i := 63; i > 0; i-- {
		if run > 0 {
			run--
			continue
		}
		s := trail[i][0]
		out[Zigzag[i]] = s.value
		run = s.from
	}
	return out
}

// TestTrellisNearWideSearch checks that mozjpeg's candidate rule loses almost
// nothing against the much slower wide search, block by block and in total.
func TestTrellisNearWideSearch(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	tables := NewQuantizationTables(85)
	configs := []TrellisConfig{DefaultTrellisConfig, {Lambda: 0.3}, {Lambda: 4, UsePerceptual: true}}
	var fast, wide float64
	for n := 0; n < 20000; n++ {
		var block [64]float32
		scale := float32(rng.IntN(4)+1) * 40
		for i := range block {
			block[i] = float32(rng.NormFloat64()) * scale / float32(1+i/4)
		}
		for _, lum := range []bool{true, false} {
			quant := tables.Luminance
			if !lum {
				quant = tables.Chrominance
			}
			config := configs[n%len(configs)]
			f := trellisCost(block, quant, config, lum, trellisOptimize(block, quant, config, lum))
			w := trellisCost(block, quant, config, lum, trellisWideSearch(block, quant, config, lum))
			if f > w*1.05+2 {
				t.Fatalf("block %d: fast cost %.2f far above wide search %.2f", n, f, w)
			}
			fast, wide = fast+f, wide+w
		}
	}
	if fast > wide*1.002 {
		t.Fatalf("total cost %.0f is more than 0.2%% above the wide search %.0f", fast, wide)
	}
	t.Logf("total cost: fast %.0f, wide %.0f (%+.3f%%)", fast, wide, (fast/wide-1)*100)
}

func BenchmarkTrellisEncode(b *testing.B) {
	img := photoLike(1024, 768, 3)
	rgb := make([]byte, 1024*768*3)
	for i := 0; i < 1024*768; i++ {
		copy(rgb[i*3:i*3+3], img.Pix[i*4:i*4+3])
	}
	opts := BalancedOptions(1024, 768, 85)
	opts.Progressive, opts.OptimizeHuffman, opts.TrellisQuant = false, false, true
	encoder, err := NewEncoder(opts)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encoder.Encode(rgb); err != nil {
			b.Fatal(err)
		}
	}
}
func TestTrellisHugeLambdaStaysFinite(t *testing.T) {
	var unit, block [64]float32
	for i := range unit {
		unit[i] = 1
	}
	block[Zigzag[1]], block[Zigzag[2]] = 1000, 1000
	for _, lambda := range []float64{1e308, math.Inf(1), math.NaN()} {
		got := trellisOptimize(block, unit, TrellisConfig{Lambda: lambda}, true)
		if got[Zigzag[1]] != 1000 || got[Zigzag[2]] != 1000 {
			t.Fatalf("lambda %g: got %d, %d; want 1000, 1000", lambda, got[Zigzag[1]], got[Zigzag[2]])
		}
	}
}
