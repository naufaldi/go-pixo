package jpeg

import (
	"math"
	"runtime"
	"testing"
)

func BenchmarkForwardDCT(b *testing.B) {
	var block [64]float32
	for i := range block {
		block[i] = float32(i*4) - 128.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ForwardDCT(block)
	}
}

func BenchmarkForwardDCTSIMD(b *testing.B) {
	var block [64]float32
	for i := range block {
		block[i] = float32(i*4) - 128.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ForwardDCTSIMD(block)
	}
}

func BenchmarkForwardDCTParallel(b *testing.B) {
	blocks := make([][64]float32, 100)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4) - 128.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ForwardDCTParallel(blocks)
	}
}

func BenchmarkForwardDCTSIMDParallel(b *testing.B) {
	blocks := make([][64]float32, 100)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4) - 128.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ForwardDCTSIMDParallel(blocks)
	}
}

func BenchmarkForwardDCTMultipleBlocks(b *testing.B) {
	blocks := make([][64]float32, 100)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4) - 128.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range blocks {
			ForwardDCT(blocks[j])
		}
	}
}

func BenchmarkForwardDCTSIMDMultipleBlocks(b *testing.B) {
	blocks := make([][64]float32, 100)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4) - 128.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range blocks {
			ForwardDCTSIMD(blocks[j])
		}
	}
}

func TestForwardDCTSIMD_VersusScalar(t *testing.T) {
	testCases := []struct {
		name  string
		block [64]float32
	}{
		{
			name:  "zeros",
			block: [64]float32{},
		},
		{
			name: "constant",
			block: func() [64]float32 {
				var b [64]float32
				for i := range b {
					b[i] = 100.0
				}
				return b
			}(),
		},
		{
			name: "step_pattern",
			block: func() [64]float32 {
				var b [64]float32
				for row := 0; row < 8; row++ {
					for col := 0; col < 8; col++ {
						if col < 4 {
							b[row*8+col] = 100
						} else {
							b[row*8+col] = -100
						}
					}
				}
				return b
			}(),
		},
		{
			name: "ramp",
			block: func() [64]float32 {
				var b [64]float32
				for i := range b {
					b[i] = float32(i) * 4
				}
				return b
			}(),
		},
		{
			name: "random",
			block: func() [64]float32 {
				var b [64]float32
				for i := range b {
					b[i] = float32(i*7+3) * float32(i%3+1) * 0.5
				}
				return b
			}(),
		},
	}

	tolerance := 1e-4

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scalarResult := ForwardDCT(tc.block)
			simdResult := ForwardDCTSIMD(tc.block)

			maxDiff := float64(0.0)
			for i := 0; i < 64; i++ {
				diff := math.Abs(float64(scalarResult[i] - simdResult[i]))
				if diff > maxDiff {
					maxDiff = diff
				}
			}

			if maxDiff > tolerance {
				t.Errorf("SIMD result differs from scalar by %f (max tolerance: %f)", maxDiff, tolerance)
			}
		})
	}
}

func TestDetectCPUFeatures(t *testing.T) {
	simd := DetectCPUFeatures()

	if simd == nil {
		t.Fatal("DetectCPUFeatures returned nil")
	}

	arch := runtime.GOARCH
	t.Logf("Architecture: %s", arch)
	t.Logf("Preferred method: %s", simd.PreferredMethod)
	t.Logf("HasSSE2: %v", simd.HasSSE2)
	t.Logf("HasSSSE3: %v", simd.HasSSSE3)
	t.Logf("HasAVX2: %v", simd.HasAVX2)
	t.Logf("HasNEON: %v", simd.HasNEON)

	switch arch {
	case "amd64":
		if !simd.HasSSE2 {
			t.Error("amd64 should have SSE2 support")
		}
		if simd.HasAVX2 || simd.HasSSSE3 || simd.HasSSE2 {
			if simd.PreferredMethod == "" {
				t.Error("PreferredMethod should not be empty on amd64")
			}
		}
	case "arm64":
		if simd.HasNEON {
			if simd.PreferredMethod == "" {
				t.Error("PreferredMethod should not be empty when NEON is available")
			}
		}
	}

	if simd.PreferredMethod == "" {
		t.Error("PreferredMethod should not be empty")
	}
}

func TestForwardDCTSIMD_GracefulFallback(t *testing.T) {
	var block [64]float32
	for i := range block {
		block[i] = float32(i*4) - 128.0
	}

	result := ForwardDCTSIMD(block)
	scalarResult := ForwardDCT(block)

	for i := range result {
		if math.Abs(float64(result[i]-scalarResult[i])) > 1e-4 {
			t.Errorf("SIMD result at index %d: got %f, want %f", i, result[i], scalarResult[i])
		}
	}
}

func TestForwardDCTSIMD_Constant(t *testing.T) {
	var block [64]float32
	for i := range block {
		block[i] = 100.0
	}

	result := ForwardDCTSIMD(block)

	if math.Abs(float64(result[0])-800.0) > 1e-3 {
		t.Errorf("DC component: got %f, want 800.0", result[0])
	}

	for i := 1; i < 64; i++ {
		if math.Abs(float64(result[i])) > 1e-3 {
			t.Errorf("AC component at %d: got %f, want 0", i, result[i])
		}
	}
}

func TestForwardDCTSIMD_Zeros(t *testing.T) {
	var block [64]float32
	result := ForwardDCTSIMD(block)

	for i, val := range result {
		if math.Abs(float64(val)) > 1e-6 {
			t.Errorf("at index %d: got %f, want 0", i, val)
		}
	}
}

func TestForwardDCTParallel_Correctness(t *testing.T) {
	blocks := make([][64]float32, 50)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4 - 128)
		}
	}

	results := ForwardDCTParallel(blocks)
	scalarResults := make([][64]float32, len(blocks))

	for i := range blocks {
		scalarResults[i] = ForwardDCT(blocks[i])
	}

	tolerance := 1e-4
	for i := range blocks {
		for j := 0; j < 64; j++ {
			diff := math.Abs(float64(results[i][j] - scalarResults[i][j]))
			if diff > tolerance {
				t.Errorf("Block %d, index %d: parallel result %f differs from scalar %f by %f",
					i, j, results[i][j], scalarResults[i][j], diff)
			}
		}
	}
}

func TestForwardDCTSIMDParallel_Correctness(t *testing.T) {
	blocks := make([][64]float32, 50)
	for i := range blocks {
		for j := range blocks[i] {
			blocks[i][j] = float32(j*4 - 128)
		}
	}

	results := ForwardDCTSIMDParallel(blocks)
	scalarResults := make([][64]float32, len(blocks))

	for i := range blocks {
		scalarResults[i] = ForwardDCT(blocks[i])
	}

	tolerance := 1e-4
	for i := range blocks {
		for j := 0; j < 64; j++ {
			diff := math.Abs(float64(results[i][j] - scalarResults[i][j]))
			if diff > tolerance {
				t.Errorf("Block %d, index %d: SIMD parallel result %f differs from scalar %f by %f",
					i, j, results[i][j], scalarResults[i][j], diff)
			}
		}
	}
}
