package png

import "math"

// CalculateEntropy computes the Shannon entropy of a byte slice.
// Entropy measures the average information content per symbol.
// Lower entropy indicates more predictable data (better for compression).
// Uses byte value frequency distribution to calculate entropy.
func CalculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0
	}

	// Count frequency of each byte value (0-255)
	freq := make([]int, 256)
	for _, b := range data {
		freq[b]++
	}

	// Calculate Shannon entropy: H = -sum(p(x) * log2(p(x)))
	// where p(x) is the probability of each symbol
	var entropy float64
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			p := float64(count) / length
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// EntropyScore calculates an estimated compressed size based on entropy.
// This provides a rough estimate of how well the data will compress.
// Lower score = better compression potential.
func EntropyScore(data []byte) float64 {
	entropy := CalculateEntropy(data)
	return entropy * float64(len(data))
}
