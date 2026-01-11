package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestZopfliIterate_Empty(t *testing.T) {
	data := []byte{}
	result, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(result))
	}
}

func TestZopfliIterate_Simple(t *testing.T) {
	data := []byte("Hello, World!")
	result, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliIterate_Repetitive(t *testing.T) {
	data := bytes.Repeat([]byte("ABC"), 100)
	result, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}
	if len(result) >= len(data) {
		t.Errorf("compression didn't reduce size: %d >= %d", len(result), len(data))
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliIterate_VersusStandard(t *testing.T) {
	enc := NewDeflateEncoder()
	data := bytes.Repeat([]byte("Hello, World! This is a test string. "), 50)

	standard, err := enc.EncodeAuto(data)
	if err != nil {
		t.Fatalf("EncodeAuto failed: %v", err)
	}

	zopfli, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}

	t.Logf("Original: %d bytes", len(data))
	t.Logf("Standard: %d bytes (%.1f%%)", len(standard), float64(len(standard))/float64(len(data))*100)
	t.Logf("Zopfli:   %d bytes (%.1f%%)", len(zopfli), float64(len(zopfli))/float64(len(data))*100)

	improvement := CalculateZopfliImprovement(standard, zopfli)
	t.Logf("Improvement over standard: %.2f%%", improvement)
}

func TestZopfliIterate_WithConfig(t *testing.T) {
	data := bytes.Repeat([]byte("Test data for compression. "), 20)

	config := NewZopfliIterationConfig()
	config.Iterations = 5
	config.BlockSplitting = true
	config.BlockSize = 65535

	result, err := ZopfliIterate(data, config)
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliIterate_IterationsImpact(t *testing.T) {
	data := bytes.Repeat([]byte("Pattern: AAAAABBBBBCCCCCDDDDD "), 30)

	config1 := NewZopfliIterationConfig()
	config1.Iterations = 1

	config5 := NewZopfliIterationConfig()
	config5.Iterations = 5

	config12 := NewZopfliIterationConfig()
	config12.Iterations = 12

	result1, err := ZopfliIterate(data, config1)
	if err != nil {
		t.Fatalf("ZopfliIterate (1 iter) failed: %v", err)
	}

	result5, err := ZopfliIterate(data, config5)
	if err != nil {
		t.Fatalf("ZopfliIterate (5 iters) failed: %v", err)
	}

	result12, err := ZopfliIterate(data, config12)
	if err != nil {
		t.Fatalf("ZopfliIterate (12 iters) failed: %v", err)
	}

	t.Logf("1 iteration:  %d bytes", len(result1))
	t.Logf("5 iterations: %d bytes", len(result5))
	t.Logf("12 iterations: %d bytes", len(result12))

	if len(result5) > len(result1) {
		t.Logf("5 iters > 1 iter: unusual")
	}
	if len(result12) > len(result5) {
		t.Logf("12 iters > 5 iters: unusual")
	}
}

func TestZopfliIterate_LargeData(t *testing.T) {
	data := make([]byte, 100000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	result, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}
	if len(result) >= len(data) {
		t.Errorf("compression didn't reduce size: %d >= %d", len(result), len(data))
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}
	if len(decompressed) != len(data) {
		t.Errorf("decompressed %d bytes, want %d", len(decompressed), len(data))
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestZopfliEncodeIterative(t *testing.T) {
	data := bytes.Repeat([]byte("Test data for iterative encoding. "), 20)

	result, err := ZopfliEncodeIterative(data, 10)
	if err != nil {
		t.Fatalf("ZopfliEncodeIterative failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed := make([]byte, len(data)*2)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestZopfliIterate_Convergence(t *testing.T) {
	data := bytes.Repeat([]byte("Convergence test data. ABCDEFGHIJKLMNOPQRSTUVWXYZ "), 50)

	var lastSize int
	convergedAt := -1

	config := NewZopfliIterationConfig()
	config.Iterations = 20
	config.ProgressCallback = func(iteration, improvement float64, size int) {
		if iteration > 3 && lastSize == size {
			if convergedAt == -1 {
				convergedAt = int(iteration)
			}
		}
		lastSize = size
	}

	_, err := ZopfliIterate(data, config)
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}

	t.Logf("Converged at iteration: %d", convergedAt)
	if convergedAt > 0 && convergedAt < 15 {
		t.Logf("Converged early as expected")
	}
}

func TestZopfliIterate_BlockSplitting(t *testing.T) {
	data := bytes.Repeat([]byte("Block splitting test with some variation. ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "), 100)

	config := NewZopfliIterationConfig()
	config.BlockSplitting = true
	config.Iterations = 10

	result, err := ZopfliIterate(data, config)
	if err != nil {
		t.Fatalf("ZopfliIterate with block splitting failed: %v", err)
	}
	if len(result) == 0 {
		t.Error("expected compressed output")
	}

	reader := flate.NewReader(bytes.NewReader(result))
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Errorf("decompressed data doesn't match original")
	}
}

func TestCalculateZopfliImprovement(t *testing.T) {
	tests := []struct {
		name      string
		original  int
		optimized int
	}{
		{"no change", 1000, 1000},
		{"50% improvement", 1000, 500},
		{"smaller output", 1000, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := make([]byte, tt.original)
			optimized := make([]byte, tt.optimized)

			improvement := CalculateZopfliImprovement(original, optimized)
			expected := (float64(tt.original) - float64(tt.optimized)) / float64(tt.original) * 100

			if improvement != expected {
				t.Errorf("got %f, want %f", improvement, expected)
			}
		})
	}
}

func TestZopfliIterate_VersusOriginalZopfli(t *testing.T) {
	data := bytes.Repeat([]byte("Comparing Zopfli implementations. Testing compression ratio and quality. "), 50)

	originalZopfli, err := ZopfliEncodeSimple(data)
	if err != nil {
		t.Fatalf("ZopfliEncodeSimple failed: %v", err)
	}

	iterative, err := ZopfliIterate(data, NewZopfliIterationConfig())
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}

	t.Logf("Original Zopfli: %d bytes (%.1f%%)", len(originalZopfli), float64(len(originalZopfli))/float64(len(data))*100)
	t.Logf("Iterative Zopfli: %d bytes (%.1f%%)", len(iterative), float64(len(iterative))/float64(len(data))*100)

	if len(iterative) < len(originalZopfli) {
		improvement := CalculateZopfliImprovement(originalZopfli, iterative)
		t.Logf("Iterative improved over original by: %.2f%%", improvement)
	}
}

func TestZopfliIterationConfig_Defaults(t *testing.T) {
	config := NewZopfliIterationConfig()

	if config.Iterations != DefaultZopfliIterations {
		t.Errorf("expected %d iterations, got %d", DefaultZopfliIterations, config.Iterations)
	}
	if config.BlockSize != DefaultBlockSize {
		t.Errorf("expected block size %d, got %d", DefaultBlockSize, config.BlockSize)
	}
	if config.SplitThreshold != DefaultSplitThreshold {
		t.Errorf("expected split threshold %f, got %f", DefaultSplitThreshold, config.SplitThreshold)
	}
	if !config.BlockSplitting {
		t.Error("expected block splitting to be enabled")
	}
}

func TestZopfliIterate_ProgressCallback(t *testing.T) {
	data := bytes.Repeat([]byte("Progress callback test. "), 20)

	var callbackIterations []float64
	var callbackImprovements []float64
	var callbackSizes []int

	config := NewZopfliIterationConfig()
	config.Iterations = 5
	config.ProgressCallback = func(iteration, improvement float64, size int) {
		callbackIterations = append(callbackIterations, iteration)
		callbackImprovements = append(callbackImprovements, improvement)
		callbackSizes = append(callbackSizes, size)
	}

	_, err := ZopfliIterate(data, config)
	if err != nil {
		t.Fatalf("ZopfliIterate failed: %v", err)
	}

	if len(callbackIterations) == 0 {
		t.Error("expected at least one callback invocation")
	}

	for i, improvement := range callbackImprovements {
		if improvement < 0 || improvement > 100 {
			t.Errorf("invalid improvement value at index %d: %f", i, improvement)
		}
	}

	for i, size := range callbackSizes {
		if size <= 0 {
			t.Errorf("invalid size at index %d: %d", i, size)
		}
	}
}
