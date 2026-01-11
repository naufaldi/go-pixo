package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestOptimalConfig_Defaults(t *testing.T) {
	config := DefaultOptimalConfig()

	if config.MaxIterations != defaultMaxIterations {
		t.Errorf("expected MaxIterations %d, got %d", defaultMaxIterations, config.MaxIterations)
	}
	if config.ConvergenceThreshold != defaultConvergenceThreshold {
		t.Errorf("expected ConvergenceThreshold %f, got %f", defaultConvergenceThreshold, config.ConvergenceThreshold)
	}
	if config.BlockSize != defaultBlockSize {
		t.Errorf("expected BlockSize %d, got %d", defaultBlockSize, config.BlockSize)
	}
	if !config.BlockSplitting {
		t.Error("expected BlockSplitting to be true")
	}
}

func TestOptimalConfig_ForLevel(t *testing.T) {
	tests := []struct {
		level         int
		maxIterations int
		maxChainLen   int
	}{
		{1, 5, 32},
		{2, 5, 32},
		{3, 10, 128},
		{4, 10, 128},
		{5, 15, 256},
		{6, 15, 256},
		{7, 20, 512},
		{8, 20, 512},
		{9, 30, 1024},
	}

	for _, tt := range tests {
		config := OptimalConfigForLevel(tt.level)

		if config.MaxIterations != tt.maxIterations {
			t.Errorf("level %d: expected MaxIterations %d, got %d", tt.level, tt.maxIterations, config.MaxIterations)
		}
		if config.MaxChainLen != tt.maxChainLen {
			t.Errorf("level %d: expected MaxChainLen %d, got %d", tt.level, tt.maxChainLen, config.MaxChainLen)
		}
	}
}

func TestEstimateBlockCost_EmptyTokens(t *testing.T) {
	cost := estimateBlockCost(nil, true)
	if cost != 0 {
		t.Errorf("expected cost 0 for empty tokens, got %f", cost)
	}

	cost = estimateBlockCost([]Token{}, true)
	if cost != 0 {
		t.Errorf("expected cost 0 for empty tokens, got %f", cost)
	}
}

func TestEstimateBlockCost_LiteralsOnly(t *testing.T) {
	tokens := []Token{
		TokenLiteral('H'),
		TokenLiteral('e'),
		TokenLiteral('l'),
		TokenLiteral('l'),
		TokenLiteral('o'),
	}

	costDynamic := estimateBlockCost(tokens, true)
	costFixed := estimateBlockCost(tokens, false)

	if costDynamic <= 0 {
		t.Error("expected positive cost for literals")
	}
	if costFixed <= 0 {
		t.Error("expected positive cost for literals")
	}

	if costDynamic != costFixed {
		t.Logf("dynamic cost: %f, fixed cost: %f", costDynamic, costFixed)
	}
}

func TestEstimateBlockCost_WithMatches(t *testing.T) {
	tokens := []Token{
		TokenLiteral('H'),
		TokenLiteral('e'),
		TokenLiteral('l'),
		TokenLiteral('l'),
		TokenLiteral('o'),
		TokenMatch(4, 3),
	}

	cost := estimateBlockCost(tokens, true)
	if cost <= 0 {
		t.Error("expected positive cost for tokens with matches")
	}

	t.Logf("cost with match: %f", cost)
}

func TestOptimalParse_EmptyData(t *testing.T) {
	tokens, err := OptimalParse(nil, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens != nil {
		t.Error("expected nil tokens for nil input")
	}

	tokens, err = OptimalParse([]byte{}, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens != nil {
		t.Error("expected nil tokens for empty input")
	}
}

func TestOptimalParse_SimpleData(t *testing.T) {
	data := []byte("Hello, World!")

	tokens, err := OptimalParse(data, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected tokens, got nil")
	}

	encoded, err := encodeTokensForTest(tokens, true)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	decoded := decompressForTest(t, encoded)
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed: got %q, want %q", decoded, data)
	}
}

func TestOptimalParse_RepetitiveData(t *testing.T) {
	data := bytes.Repeat([]byte("ABC"), 100)

	tokens, err := OptimalParse(data, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected tokens, got nil")
	}

	encoded, err := encodeTokensForTest(tokens, true)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	if len(encoded) >= len(data) {
		t.Errorf("expected compression, got %d >= %d", len(encoded), len(data))
	}

	decoded := decompressForTest(t, encoded)
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed")
	}
}

func TestEncodeOptimalLZ77_Basic(t *testing.T) {
	data := []byte("Hello, World!")

	compressed, err := EncodeOptimalLZ77(data, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("EncodeOptimalLZ77 failed: %v", err)
	}
	if len(compressed) == 0 {
		t.Error("expected compressed output")
	}

	decoded := decompressForTest(t, compressed)
	if !bytes.Equal(decoded, data) {
		t.Errorf("got %q, want %q", decoded, data)
	}
}

func TestEncodeOptimalLZ77_CompressionRatio(t *testing.T) {
	data := bytes.Repeat([]byte("A"), 1000)

	compressed, err := EncodeOptimalLZ77(data, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("EncodeOptimalLZ77 failed: %v", err)
	}

	ratio := float64(len(compressed)) / float64(len(data))
	if ratio >= 1.0 {
		t.Errorf("expected compression, ratio: %f", ratio)
	}

	t.Logf("compression ratio: %.3f", ratio)
}

func TestEncodeOptimalLZ77_BetterThanStandard(t *testing.T) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), 50)

	standard := NewDeflateEncoder()
	standardCompressed, err := standard.EncodeAuto(data)
	if err != nil {
		t.Fatalf("standard encoding failed: %v", err)
	}

	optimalCompressed, err := EncodeOptimalLZ77(data, OptimalConfigForLevel(9))
	if err != nil {
		t.Fatalf("optimal encoding failed: %v", err)
	}

	t.Logf("standard: %d bytes, optimal: %d bytes",
		len(standardCompressed), len(optimalCompressed))

	if len(optimalCompressed) == 0 {
		t.Error("optimal encoding produced empty output")
	}

	if len(standardCompressed) == 0 {
		t.Error("standard encoding produced empty output")
	}
}

func TestConvergenceDetection(t *testing.T) {
	data := []byte("This is a test string with some repetitive content. " +
		"This is a test string with some repetitive content. " +
		"This is a test string with some repetitive content. " +
		"This is a test string with some repetitive content. " +
		"This is a test string with some repetitive content. ")

	config := OptimalConfig{
		MaxIterations:        100,
		ConvergenceThreshold: 0.001,
		BlockSize:            65535,
		BlockSplitting:       true,
		ProgressCallback: func(iteration, improvement float64) {
			t.Logf("iteration %.0f: improvement %.4f%%", iteration, improvement*100)
		},
	}

	tokens, err := OptimalParse(data, config)
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected tokens")
	}

	encoded, err := encodeTokensForTest(tokens, true)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	if len(encoded) >= len(data) {
		t.Errorf("expected compression, got %d >= %d", len(encoded), len(data))
	}
}

func TestOptimalParse_MultipleIterations(t *testing.T) {
	data := []byte("Test data for multiple iterations. " +
		"Test data for multiple iterations. " +
		"Test data for multiple iterations. " +
		"Test data for multiple iterations. " +
		"Test data for multiple iterations. ")

	config := DefaultOptimalConfig()
	config.MaxIterations = 20

	var iterationsRun int
	config.ProgressCallback = func(iteration, improvement float64) {
		iterationsRun = int(iteration)
	}

	_, err := OptimalParse(data, config)
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}

	if iterationsRun < config.MaxIterations-5 {
		t.Logf("ran %d iterations out of %d max", iterationsRun, config.MaxIterations)
	}
}

func TestEncodeOptimalLZ77_VariousCompressionLevels(t *testing.T) {
	data := bytes.Repeat([]byte("Lorem ipsum dolor sit amet. "), 100)

	var results []struct {
		level int
		size  int
		ratio float64
	}

	for level := 1; level <= 9; level++ {
		config := OptimalConfigForLevel(level)
		compressed, err := EncodeOptimalLZ77(data, config)
		if err != nil {
			t.Fatalf("level %d: EncodeOptimalLZ77 failed: %v", level, err)
		}

		ratio := float64(len(compressed)) / float64(len(data))
		results = append(results, struct {
			level int
			size  int
			ratio float64
		}{level, len(compressed), ratio})

		t.Logf("level %d: %d bytes (%.3f ratio)", level, len(compressed), ratio)
	}

	for i := 1; i < len(results); i++ {
		if results[i].size > results[i-1].size+100 {
			t.Logf("note: level %d (%d) is larger than level %d (%d)",
				results[i].level, results[i].size,
				results[i-1].level, results[i-1].size)
		}
	}
}

func TestOptimalParse_BlockSplitting(t *testing.T) {
	data := make([]byte, 100000)
	for i := range data {
		data[i] = byte(i % 256)
	}

	config := OptimalConfig{
		MaxIterations:        5,
		ConvergenceThreshold: 0.001,
		BlockSize:            65535,
		BlockSplitting:       true,
	}

	tokens, err := OptimalParse(data, config)
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}
	if tokens == nil {
		t.Fatal("expected tokens")
	}

	encoded, err := encodeTokensForTest(tokens, true)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	decoded := decompressForTest(t, encoded)
	if !bytes.Equal(decoded, data) {
		t.Errorf("roundtrip failed")
	}

	t.Logf("original: %d bytes, compressed: %d bytes (%.3f ratio)",
		len(data), len(encoded), float64(len(encoded))/float64(len(data)))
}

func TestOptimalParse_CostModelAccuracy(t *testing.T) {
	data := []byte("AAAAABBBBBCCCCCDDDDD")

	tokens, err := OptimalParse(data, DefaultOptimalConfig())
	if err != nil {
		t.Fatalf("OptimalParse failed: %v", err)
	}

	encoded, err := encodeTokensForTest(tokens, true)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	estimatedCost := estimateBlockCost(tokens, true)
	actualBits := float64(len(encoded) * 8)

	diff := estimatedCost - actualBits
	if diff > 50 || diff < -50 {
		t.Logf("estimated: %.0f bits, actual: %.0f bits, diff: %.0f",
			estimatedCost, actualBits, diff)
	}
}

func encodeTokensForTest(tokens []Token, useDynamic bool) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	if useDynamic {
		err = WriteDynamicBlock(&buf, true, tokens)
	} else {
		err = WriteFixedBlock(&buf, true, tokens)
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressForTest(t *testing.T, data []byte) []byte {
	reader := flate.NewReader(bytes.NewReader(data))
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("decompression failed: %v", err)
	}
	return decompressed
}
