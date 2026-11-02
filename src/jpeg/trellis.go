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

	// A pending run of r zeros at position i can only come from the nonzero
	// state at position j = i-r, followed by zeros at j+1..i. With nonzero[j]
	// that state's cost and zeroSum[i] the summed zero distortion of positions
	// 1..i, the run state costs base[j] + zeroSum[i], where
	// base[j] = nonzero[j] - zeroSum[j]. That removes the per-position shift of
	// every run state and makes the cheapest predecessor an O(1) running min.
	var base [64]float64 // base[0] is the DC position: cost 0, no zeros yet.
	var predecessors [64]trellisPredecessor
	rates := &acRates[0]
	if !isLuminance {
		rates = &acRates[1]
	}
	zrlRate := rates.byRun[0][15]
	eobRate := rates.byRun[0][0]
	zeroSum, minBase := 0.0, 0.0
	for i := 1; i < 64; i++ {
		index := Zigzag[i]
		previousZeroSum := zeroSum
		zeroSum += config.Lambda * calculateDistortion(dct[index], 0, index, config.UsePerceptual)
		minPrevious := minBase + previousZeroSum

		best := math.Inf(1)
		var bestState trellisPredecessor
		candidates, count := generateCandidates(dct[index], quantTable[index], -1023, 1023)
		for _, value := range candidates[:count] {
			if value == 0 {
				continue
			}
			distortion := config.Lambda * calculateDistortion(dct[index], float32(value)*quantTable[index], index, config.UsePerceptual)
			category := getACCategory(value)
			byRun := &rates.byRun[category]
			// Rates are non-negative and float addition is monotonic, so when
			// even the cheapest run cannot beat the current best, none can.
			if minPrevious+float64(rates.minimum[category])+distortion >= best {
				continue
			}
			for runLength := 0; runLength < i; runLength++ {
				rate := (runLength/16)*zrlRate + byRun[runLength%16]
				cost := base[i-1-runLength] + previousZeroSum + float64(rate) + distortion
				if cost < best {
					best = cost
					bestState = trellisPredecessor{quantizedVal: value, prevIndex: uint8(runLength)}
				}
			}
		}
		predecessors[i] = bestState
		base[i] = best - zeroSum
		minBase = math.Min(minBase, base[i])
	}

	// State 0 at the last position needs no EOB; any trailing run needs one,
	// even when it is longer than 16.
	bestIndex := 0
	bestCost := base[63] + zeroSum
	for runLength := 1; runLength < 64; runLength++ {
		cost := base[63-runLength] + zeroSum + float64(eobRate)
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
// Values come nearest-first, so the trellis finds a good cost early and its
// lower bound skips the far candidates without scanning every run length.
func generateCandidates(coeff, step float32, minVal, maxVal int) ([8]int16, int) {
	centerVal := int(math.Round(float64(coeff / step)))
	var candidates [8]int16
	count := 0
	for _, delta := range [...]int{0, -1, 1, -2, 2, -3, 3} {
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

// acRateTable caches estimateACRate for the fixed standard tables:
// byRun[category][run], and the smallest rate per category over all runs.
type acRateTable struct {
	byRun   [16][16]int
	minimum [16]int
}

// acRates holds luminance ([0]) and chrominance ([1]) rates. The trellis inner
// loop runs O(64²) times per block, so table lookups replace Huffman calls.
var acRates = func() (tables [2]acRateTable) {
	for t, isLuminance := range []bool{true, false} {
		for category := 0; category < 16; category++ {
			tables[t].minimum[category] = math.MaxInt
			for run := 0; run < 16; run++ {
				rate := estimateACRate(uint8(run), uint8(category), isLuminance)
				tables[t].byRun[category][run] = rate
				tables[t].minimum[category] = min(tables[t].minimum[category], rate)
			}
		}
	}
	return tables
}()

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
