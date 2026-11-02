package jpeg

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"testing"
)

// trellisGoldenHash hashes trellis output for deterministic pseudo-random
// blocks across both table types and several configurations.
func trellisGoldenHash(optimize func([64]float32, [64]float32, TrellisConfig, bool) [64]int16) uint64 {
	rng := rand.New(rand.NewPCG(1, 2))
	qt := NewQuantizationTables(85)
	hash := fnv.New64a()
	var buf [2]byte
	for n := 0; n < 20000; n++ {
		var block [64]float32
		scale := float32(rng.IntN(4)+1) * 40
		for i := range block {
			block[i] = float32(rng.NormFloat64()) * scale / float32(1+i/4)
		}
		for _, lum := range []bool{true, false} {
			q := qt.Luminance
			if !lum {
				q = qt.Chrominance
			}
			for _, cfg := range []TrellisConfig{DefaultTrellisConfig, {Lambda: 0.3}, {Lambda: 4, UsePerceptual: true}} {
				for _, v := range optimize(block, q, cfg, lum) {
					binary.LittleEndian.PutUint16(buf[:], uint16(v))
					hash.Write(buf[:])
				}
			}
		}
	}
	return hash.Sum64()
}

// TestTrellisMatchesReferenceOutput pins trellis output to the hash produced
// by the original O(64²)-per-candidate implementation (master 2c3dfc7) before it was made
// faster (prefix-sum run states, cached rates, nearest-first pruning).
// Faster must not mean different.
func TestTrellisMatchesReferenceOutput(t *testing.T) {
	const reference uint64 = 0xb83bc8a6cea9591d
	if got := trellisGoldenHash(trellisOptimize); got != reference {
		t.Fatalf("trellis output hash %#x, want %#x", got, reference)
	}
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

// trellisTieBlocks are deterministic blocks whose candidates tie exactly:
// coefficients sit on half quantization steps, alone and in pairs.
func trellisTieBlocks() (blocks [][64]float32, quants [][64]float32) {
	var unit [64]float32
	for i := range unit {
		unit[i] = 1
	}
	std := NewQuantizationTables(85).Luminance
	for _, q := range [][64]float32{unit, std} {
		for pos := 1; pos < 64; pos += 3 {
			for _, half := range []float32{0.5, 1.5, 2.5, 3.5, 10.5, -2.5} {
				var block [64]float32
				block[Zigzag[pos]] = half * q[Zigzag[pos]]
				blocks, quants = append(blocks, block), append(quants, q)
				if pos+5 < 64 {
					block[Zigzag[pos+5]] = -half * q[Zigzag[pos+5]]
					blocks, quants = append(blocks, block), append(quants, q)
				}
			}
		}
	}
	return blocks, quants
}

// TestTrellisKeepsTieBreaking pins exact-tie behavior to the original
// ascending-candidate search: equal costs pick the smaller value, then the
// shorter run, regardless of the order candidates are evaluated in.
func TestTrellisKeepsTieBreaking(t *testing.T) {
	const reference uint64 = 0xbc32d6f91f8de5c6
	hash := fnv.New64a()
	var buf [2]byte
	blocks, quants := trellisTieBlocks()
	for n, block := range blocks {
		for _, lum := range []bool{true, false} {
			for _, cfg := range []TrellisConfig{DefaultTrellisConfig, {Lambda: 1}, {Lambda: 100}, {Lambda: 100, UsePerceptual: true}, {Lambda: 0.01}} {
				for _, v := range trellisOptimize(block, quants[n], cfg, lum) {
					binary.LittleEndian.PutUint16(buf[:], uint16(v))
					hash.Write(buf[:])
				}
			}
		}
	}
	if got := hash.Sum64(); got != reference {
		t.Fatalf("tie-case output hash %#x, want %#x", got, reference)
	}

	// 2 and 3 cost the same for 2.5 on a unit step; the original picks 2.
	var unit, block [64]float32
	for i := range unit {
		unit[i] = 1
	}
	block[Zigzag[1]] = 2.5
	if got := trellisOptimize(block, unit, TrellisConfig{Lambda: 100}, true)[Zigzag[1]]; got != 2 {
		t.Fatalf("tie resolved to %d, want 2", got)
	}
}

// TestTrellisHugeLambdaStaysFinite: an unbounded weight once overflowed the
// prefix sums to -Inf/NaN and silently dropped every later coefficient.
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
