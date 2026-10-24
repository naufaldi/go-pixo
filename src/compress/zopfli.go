package compress

import "math"

// ZopfliConfig controls a bounded search over greedy LZ77 parses, not full Zopfli.
type ZopfliConfig struct {
	// Iterations budgets distinct trials beyond the always-retained level-6 parse.
	// One trial tries level 9; two also try level 7; three also try level 8.
	// Larger values are capped at three because no further strategies are implemented.
	// Nonpositive values use the legacy default of 15 (effectively three trials).
	Iterations int

	// BlockSplitting is retained for compatibility but is unsupported and ignored.
	// Output uses a single fixed or dynamic Huffman block.
	BlockSplitting bool

	// MaxBlockSize is retained for compatibility but is unsupported and ignored.
	// The 65535-byte limit applies only to stored blocks, not Huffman-coded blocks.
	MaxBlockSize int

	// ProgressCallback reports completed distinct trials after the baseline parse.
	// total is the capped trial count, not the originally requested Iterations.
	ProgressCallback func(iteration, total int)
}

// DefaultZopfliConfig returns the default Zopfli configuration.
func DefaultZopfliConfig() ZopfliConfig {
	return ZopfliConfig{
		Iterations:     15,
		BlockSplitting: true,
		MaxBlockSize:   65535,
	}
}

// ZopfliEncode selects the smallest valid fixed/dynamic encoding across distinct
// greedy parses. Tokens are generated once per configuration and shared by both
// Huffman coders. This does not implement Zopfli's iterative cost-model refinement
// or block splitting. Empty input retains the historical empty-result behavior.
func ZopfliEncode(data []byte, config ZopfliConfig) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	if config.Iterations <= 0 {
		config.Iterations = DefaultZopfliConfig().Iterations
	}
	encoder := NewDeflateEncoder()

	// Always start with a compressed result, never return uncompressed data
	bestResult, err := encoder.EncodeAuto(data)
	if err != nil {
		return nil, err
	}

	// Try level 9 first to preserve the old minimum-effort quality. Intermediate
	// chain limits can find different greedy matches, so higher budgets add 7 and 8.
	levels := [...]int{9, 7, 8}
	trials := min(config.Iterations, len(levels))
	for iteration, level := range levels[:trials] {
		encoder.SetCompressionLevel(level)
		candidate, err := encoder.EncodeAuto(data)
		if err != nil {
			return nil, err
		}
		if len(candidate) < len(bestResult) {
			bestResult = candidate
		}

		if config.ProgressCallback != nil {
			config.ProgressCallback(iteration+1, trials)
		}
	}

	return bestResult, nil
}

// ZopfliEncodeSimple is a simplified Zopfli encoder with reasonable defaults.
// The legacy default budget is capped to the three supported distinct trials.
func ZopfliEncodeSimple(data []byte) ([]byte, error) {
	return ZopfliEncode(data, DefaultZopfliConfig())
}

// CalculateDeflateCost estimates the compressed size of data using a cost model.
// This is used by compression algorithms to evaluate different configurations.
// Lower cost = better compression potential.
func CalculateDeflateCost(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Count literal frequencies for cost estimation
	freq := make([]int, 288) // 0-255 literals + 256-285 length codes
	for _, b := range data {
		freq[b]++
	}

	// Add end-of-block code
	freq[256]++

	// Estimate Huffman code cost using Shannon entropy approximation
	cost := 0.0
	total := float64(len(data))

	for i := 0; i < 288; i++ {
		if freq[i] > 0 {
			p := float64(freq[i]) / total
			// Approximate cost: p * log2(1/p) with small overhead
			cost += p * (math.Log2(float64(total)/float64(freq[i])) + 0.5)
		}
	}

	// Add overhead for distance codes and block structure
	cost += float64(len(data)) * 0.01

	return cost
}
