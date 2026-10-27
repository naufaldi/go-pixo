package jpeg

import (
	"math"
)

var huffmanTables = NewHuffmanTables()

var (
	visualSensitivityWeights = [64]float64{
		1.00, 0.80, 0.70, 0.60, 0.55, 0.50, 0.45, 0.40,
		0.75, 0.70, 0.65, 0.60, 0.55, 0.50, 0.45, 0.40,
		0.65, 0.60, 0.55, 0.50, 0.50, 0.45, 0.40, 0.40,
		0.60, 0.55, 0.50, 0.50, 0.45, 0.45, 0.40, 0.40,
		0.55, 0.50, 0.45, 0.45, 0.40, 0.40, 0.40, 0.35,
		0.50, 0.45, 0.45, 0.40, 0.40, 0.40, 0.35, 0.35,
		0.45, 0.45, 0.40, 0.40, 0.35, 0.35, 0.35, 0.35,
		0.40, 0.40, 0.40, 0.35, 0.35, 0.35, 0.35, 0.35,
	}
)

// TrellisConfig tunes rate-distortion trellis quantization of AC coefficients.
type TrellisConfig struct {
	// Lambda weights distortion in the cost: rate + Lambda*distortion.
	Lambda        float64
	MaxIterations int
	UsePerceptual bool
}

// DefaultTrellisConfig is the trellis configuration used when callers only supply a lambda.
var DefaultTrellisConfig = TrellisConfig{
	Lambda:        1.0,
	MaxIterations: 64,
	UsePerceptual: true,
}

// PerceptualDistortion returns the squared error between dct and quantized, weighted by per-frequency visual sensitivity.
func PerceptualDistortion(dct, quantized [64]float32) float64 {
	totalDistortion := 0.0
	for i := 0; i < 64; i++ {
		diff := float64(dct[i] - quantized[i])
		weightedDistortion := diff * diff * visualSensitivityWeights[i]
		totalDistortion += weightedDistortion
	}
	return totalDistortion
}

// EstimateSymbolRate estimates the bits needed to entropy-code a quantized block with the standard Huffman tables.
func EstimateSymbolRate(coeffs [64]int16, isLuminance bool) int {
	totalBits := 0

	dcCategory := getDCCategory(coeffs[0])
	dcCode, dcLen := huffmanTables.EncodeDC(uint8(dcCategory), isLuminance)
	_ = dcCode
	totalBits += int(dcLen)
	if coeffs[0] != 0 {
		totalBits += dcCategory
	}

	runLength := 0
	for i := 1; i < 64; i++ {
		if coeffs[i] == 0 {
			runLength++
			continue
		}

		for runLength >= 16 {
			acCode, acLen := huffmanTables.EncodeAC(15, 0, isLuminance)
			_ = acCode
			totalBits += int(acLen)
			runLength -= 16
		}

		acCategory := getACCategory(coeffs[i])
		acCode, acLen := huffmanTables.EncodeAC(uint8(runLength), uint8(acCategory), isLuminance)
		_ = acCode
		totalBits += int(acLen)
		totalBits += acCategory

		runLength = 0
	}

	if runLength > 0 {
		acCode, acLen := huffmanTables.EncodeAC(uint8(runLength), 0, isLuminance)
		_ = acCode
		totalBits += int(acLen)
	}

	return totalBits
}

func getDCCategory(val int16) int {
	absVal := absInt16(val)
	if absVal == 0 {
		return 0
	}
	category := 0
	for absVal > 0 {
		category++
		absVal >>= 1
	}
	return category
}

func getACCategory(val int16) int {
	absVal := absInt16(val)
	if absVal == 0 {
		return 0
	}
	category := 0
	for absVal > 0 {
		category++
		absVal >>= 1
	}
	return category
}

// A state is indexed by its pending zero-run length. All nonzero candidates
// reach state zero, so only its winning value and predecessor need storing.
// A zero transition always comes from state (runLength - 1).
type trellisPredecessor struct {
	quantizedVal int16
	prevIndex    uint8
}

// TrellisOptimizeFull runs luminance trellis quantization using DefaultTrellisConfig with the given lambda.
func TrellisOptimizeFull(dct [64]float32, quantTable [64]float32, lambda float64) [64]int16 {
	config := DefaultTrellisConfig
	config.Lambda = lambda
	return TrellisOptimizeWithConfig(dct, quantTable, config)
}

// TrellisOptimizeWithConfig optimizes AC coefficients using luminance rates.
// Inputs and output use natural order; rate estimation follows JPEG zigzag order.
// DC uses ordinary quantization and is never part of the trellis search.
func TrellisOptimizeWithConfig(dct [64]float32, quantTable [64]float32, config TrellisConfig) [64]int16 {
	return trellisOptimize(dct, quantTable, config, true)
}

func trellisOptimize(dct [64]float32, quantTable [64]float32, config TrellisConfig, isLuminance bool) [64]int16 {
	if config.Lambda == 0 {
		config.Lambda = DefaultTrellisConfig.Lambda
	}

	var result [64]int16
	result[0] = int16(math.Round(float64(dct[0] / quantTable[0])))

	// At scan position i there are i+1 possible pending runs, including zero.
	// Costs roll between two fixed buffers; predecessors survive for backtracking.
	// No state/candidate slices or pointers escape per block or transition.
	var costs [2][64]float64
	var predecessors [64]trellisPredecessor
	zrlRate := estimateACRate(15, 0, isLuminance)
	eobRate := estimateACRate(0, 0, isLuminance)
	for i := 1; i < 64; i++ {
		index := Zigzag[i]
		previous, current := &costs[(i-1)%2], &costs[i%2]
		current[0] = math.Inf(1)

		zeroCost := config.Lambda * calculateDistortion(dct[index], 0, index, config.UsePerceptual)
		for runLength := 0; runLength < i; runLength++ {
			// Defer zero-run rate until a nonzero value or EOB is emitted.
			current[runLength+1] = previous[runLength] + zeroCost
		}

		candidates, count := generateCandidates(dct[index], quantTable[index], -1023, 1023)
		for _, value := range candidates[:count] {
			if value == 0 {
				continue
			}
			distortion := config.Lambda * calculateDistortion(dct[index], float32(value)*quantTable[index], index, config.UsePerceptual)
			category := uint8(getACCategory(value))
			for runLength := 0; runLength < i; runLength++ {
				rate := (runLength/16)*zrlRate + estimateACRate(uint8(runLength%16), category, isLuminance)
				cost := previous[runLength] + float64(rate) + distortion
				if cost < current[0] {
					current[0] = cost
					predecessors[i] = trellisPredecessor{
						quantizedVal: value,
						prevIndex:    uint8(runLength),
					}
				}
			}
		}
	}

	bestIndex := 0
	bestCost := costs[1][0]
	for runLength := 1; runLength < 64; runLength++ {
		// Trailing zeros use one EOB, even for runs longer than 16.
		cost := costs[1][runLength] + float64(eobRate)
		if cost < bestCost {
			bestCost, bestIndex = cost, runLength
		}
	}
	for i := 63; i > 0; i-- {
		if bestIndex > 0 {
			bestIndex--
			continue
		}
		state := predecessors[i]
		result[Zigzag[i]] = state.quantizedVal
		bestIndex = int(state.prevIndex)
	}
	return result
}

// generateCandidates includes ordinary rounding, its nearby alternatives and
// zero. Baseline 8-bit JPEG permits AC magnitudes up to 1023, not just 255.
func generateCandidates(coeff, step float32, minVal, maxVal int) ([8]int16, int) {
	centerVal := int(math.Round(float64(coeff / step)))
	var candidates [8]int16
	count := 0
	for delta := -3; delta <= 3; delta++ {
		candidate := centerVal + delta
		if candidate != 0 && candidate >= minVal && candidate <= maxVal {
			candidates[count] = int16(candidate)
			count++
		}
	}
	// Zero is represented by the pending-run transition in the optimizer.
	candidates[count] = 0
	return candidates, count + 1
}

func calculateDistortion(original, quantized float32, index int, usePerceptual bool) float64 {
	if usePerceptual {
		return calculatePerceptualDistortion(original, quantized, index)
	}
	return calculateSquaredDistortion(original, quantized)
}

func calculatePerceptualDistortion(original, quantized float32, index int) float64 {
	diff := float64(original - quantized)
	return diff * diff * visualSensitivityWeights[index]
}

func calculateSquaredDistortion(original, quantized float32) float64 {
	diff := float64(original - quantized)
	return diff * diff
}

func estimateDCRate(quantVal int16, isLuminance bool) int {
	category := getDCCategory(quantVal)
	_, length := huffmanTables.EncodeDC(uint8(category), isLuminance)
	bits := int(length)
	if quantVal != 0 {
		bits += category
	}
	return bits
}

func estimateACRate(runLength, category uint8, isLuminance bool) int {
	_, length := huffmanTables.EncodeAC(runLength, category, isLuminance)
	bits := int(length)
	if category > 0 {
		bits += int(category)
	}
	return bits
}

// TrellisQuantize runs luminance trellis quantization with the given lambda, 64 iterations, and perceptual weighting.
func TrellisQuantize(dct [64]float32, quantTable [64]float32, lambda float32) [64]int16 {
	config := TrellisConfig{
		Lambda:        float64(lambda),
		MaxIterations: 64,
		UsePerceptual: true,
	}
	return TrellisOptimizeWithConfig(dct, quantTable, config)
}

func quantizeSingleCoefficient(coeff float32, quant float32, minVal, maxVal int, lambda float32) int16 {
	step := quant

	centerVal := int16(math.Round(float64(coeff / step)))

	bestVal := centerVal
	bestCost := calculateCost(float32(centerVal)*step, coeff, 0, lambda)

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

func calculateRate(quantVal int16) float32 {
	absVal := absInt16(quantVal)

	category := uint8(0)
	temp := absVal
	for temp > 0 {
		category++
		temp >>= 1
	}

	baseCost := float32(category)

	if absVal > 0 {
		valueCost := float32(category)
		return baseCost + valueCost
	}

	return baseCost
}

func calculateCost(quantizedVal, originalVal, baseRate float32, lambda float32) float32 {
	distortion := (originalVal - quantizedVal) * (originalVal - quantizedVal)
	rate := baseRate
	return rate + lambda*distortion
}

func absInt16(v int16) int16 {
	if v < 0 {
		return -v
	}
	return v
}

// CalculateLambda maps a 1-100 quality to the trellis rate-distortion weight, giving lower quality a larger lambda.
func CalculateLambda(quality uint8) float32 {
	qualityFloat := float32(quality)
	if qualityFloat >= 50 {
		return 0.1 + (100-qualityFloat)/50*0.9
	}
	return 1.0 + (50-qualityFloat)/50*4.0
}
