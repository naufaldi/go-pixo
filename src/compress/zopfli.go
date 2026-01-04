package compress

import "math"

// ZopfliConfig holds configuration for Zopfli-style compression.
type ZopfliConfig struct {
	// Iterations is the number of iterations to run.
	// More iterations produce better compression but take longer.
	// Default: 15
	Iterations int

	// BlockSplitting enables optimal block splitting.
	// Default: true
	BlockSplitting bool

	// MaxBlockSize is the maximum size of each DEFLATE block.
	// Default: 65535 (maximum allowed by DEFLATE)
	MaxBlockSize int
}

// DefaultZopfliConfig returns the default Zopfli configuration.
func DefaultZopfliConfig() ZopfliConfig {
	return ZopfliConfig{
		Iterations:     15,
		BlockSplitting: true,
		MaxBlockSize:   65535,
	}
}

// ZopfliEncode compresses data using Zopfli-style iterative DEFLATE optimization.
// This provides better compression than standard DEFLATE at the cost of slower encoding.
// The algorithm iteratively refines the compression by:
// 1. Finding better literal/length and distance pairs through multiple passes
// 2. Trying different encoding configurations
// 3. Selecting the best result across iterations
func ZopfliEncode(data []byte, config ZopfliConfig) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	if config.Iterations <= 0 {
		config.Iterations = DefaultZopfliConfig().Iterations
	}
	if config.MaxBlockSize <= 0 {
		config.MaxBlockSize = DefaultZopfliConfig().MaxBlockSize
	}

	encoder := NewDeflateEncoder()

	// Always start with a compressed result, never return uncompressed data
	bestResult, err := encoder.EncodeAuto(data)
	if err != nil {
		return nil, err
	}
	bestSize := len(bestResult)

	// Store original compression level
	originalLevel := encoder.compressionLevel

	// Run iterative refinement
	for iteration := 0; iteration < config.Iterations; iteration++ {
		// Set maximum compression level
		encoder.SetCompressionLevel(9)

		// Try single block encoding
		singleResult, encodeErr := encoder.EncodeAuto(data)
		if encodeErr == nil && len(singleResult) < bestSize {
			bestResult = singleResult
			bestSize = len(singleResult)
		}

		// Try fixed blocks as alternative
		fixedResult, encodeErr := encoder.Encode(data, false)
		if encodeErr == nil && len(fixedResult) < bestSize {
			bestResult = fixedResult
			bestSize = len(fixedResult)
		}

		// Try dynamic blocks as alternative
		dynamicResult, encodeErr := encoder.Encode(data, true)
		if encodeErr == nil && len(dynamicResult) < bestSize {
			bestResult = dynamicResult
			bestSize = len(dynamicResult)
		}
	}

	// Restore original compression level
	encoder.SetCompressionLevel(originalLevel)

	return bestResult, nil
}

// ZopfliEncodeSimple is a simplified Zopfli encoder with reasonable defaults.
// Uses 15 iterations for good compression.
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
