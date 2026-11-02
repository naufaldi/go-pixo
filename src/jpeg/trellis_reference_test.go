package jpeg

import (
	"encoding/binary"
	"hash/fnv"
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
// by the original O(64²)-per-candidate implementation before it was made
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
