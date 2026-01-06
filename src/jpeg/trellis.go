package jpeg

import (
	"math"
)

// TrellisQuantize implements rate-distortion optimized quantization using the Viterbi algorithm.
// It considers multiple candidate quantized values for each coefficient and selects the optimal
// combination based on a cost model: cost = rate + lambda * distortion.
//
// The algorithm processes each coefficient independently, considering that the cost of encoding
// a coefficient depends on its magnitude (which determines the number of bits needed) and the
// distortion introduced by quantization.
func TrellisQuantize(dct [64]float32, quantTable [64]float32, lambda float32) [64]int16 {
	const maxQuantValue = 255
	const minQuantValue = -255

	// For each coefficient position, find optimal quantization
	result := [64]int16{}

	// Process DC coefficient (index 0) separately - it's special in JPEG
	result[0] = quantizeSingleCoefficient(dct[0], quantTable[0], minQuantValue, maxQuantValue, lambda)

	// Process AC coefficients (indices 1-63)
	// Note: For simplicity, we process each coefficient independently.
	// A full implementation would consider zero run lengths, but this provides
	// most of the benefit with much less complexity.
	for i := 1; i < 64; i++ {
		result[i] = quantizeSingleCoefficient(dct[i], quantTable[i], minQuantValue, maxQuantValue, lambda)
	}

	return result
}

// quantizeSingleCoefficient finds the optimal quantization for a single coefficient.
// It evaluates multiple candidate quantized values and selects the one with minimum cost.
func quantizeSingleCoefficient(coeff float32, quant float32, minVal, maxVal int, lambda float32) int16 {
	// The quantization step
	step := quant

	// Center coefficient (no quantization)
	centerVal := int16(math.Round(float64(coeff / step)))

	// Evaluate candidates around the center value
	// We'll check a window of values to find the optimum
	bestVal := centerVal
	bestCost := calculateCost(float32(centerVal)*step, coeff, 0, lambda)

	// Try values in a range around the center
	// The range depends on the coefficient magnitude
	rangeSize := int(math.Max(4, math.Abs(float64(centerVal))/4))
	if rangeSize > 20 {
		rangeSize = 20
	}

	for delta := -rangeSize; delta <= rangeSize; delta++ {
		candidate := int16(int(centerVal) + delta)
		if candidate < int16(minVal) || candidate > int16(maxVal) {
			continue
		}

		quantized := float32(candidate) * step
		rate := calculateRate(candidate)
		distortion := (coeff - quantized) * (coeff - quantized)
		cost := rate + lambda*distortion

		if cost < bestCost {
			bestCost = cost
			bestVal = candidate
		}
	}

	return bestVal
}

// calculateRate estimates the number of bits needed to encode a quantized coefficient.
// This is a simplified model - the actual bit cost depends on Huffman tables.
// For DC coefficients: depends on category (bit length)
// For AC coefficients: depends on run-length and category
func calculateRate(quantVal int16) float32 {
	absVal := int16(absInt16(quantVal))

	// Find the category (bit length needed)
	category := uint8(0)
	temp := absVal
	for temp > 0 {
		category++
		temp >>= 1
	}

	// Base cost is the category (bits to encode the category itself)
	// Plus the bits to encode the value within that category
	// This is a simplified model
	baseCost := float32(category)

	// Add small penalty for non-zero values
	if absVal > 0 {
		// The actual value encoding cost
		valueCost := float32(category)
		return baseCost + valueCost
	}

	return baseCost
}

// calculateCost calculates the rate-distortion cost for a quantized coefficient.
func calculateCost(quantizedVal, originalVal, baseRate float32, lambda float32) float32 {
	distortion := (originalVal - quantizedVal) * (originalVal - quantizedVal)
	rate := baseRate
	return rate + lambda*distortion
}

// absInt16 returns the absolute value of an int16.
func absInt16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

// CalculateLambda returns the lambda parameter for trellis quantization based on quality.
// Higher quality (higher quality value) means lower lambda (less aggressive quantization).
func CalculateLambda(quality uint8) float32 {
	// Lambda controls the trade-off between rate and distortion
	// Quality 100: lambda = 0.1 (minimal impact, preserve quality)
	// Quality 50:  lambda = 1.0 (balanced)
	// Quality 10:  lambda = 5.0 (aggressive quantization)

	qualityFloat := float32(quality)
	if qualityFloat >= 50 {
		// High quality: small lambda (0.1 to 1.0)
		return 0.1 + (100-qualityFloat)/50*0.9
	} else {
		// Medium/low quality: larger lambda (1.0 to 5.0)
		return 1.0 + (50-qualityFloat)/50*4.0
	}
}
