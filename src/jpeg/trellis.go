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

// maxTrellisLambda bounds the distortion weight (see trellisOptimize).
const maxTrellisLambda = 1e6

func trellisOptimize(dct [64]float32, quantTable [64]float32, config TrellisConfig, isLuminance bool) [64]int16 {
	if config.Lambda == 0 {
		config.Lambda = DefaultTrellisConfig.Lambda
	}
	// Costs are kept as prefix sums; a bounded weight keeps them finite for any
	// 8-bit DCT input. Larger weights make rate negligible anyway.
	if !(config.Lambda <= maxTrellisLambda) {
		config.Lambda = maxTrellisLambda
	}

	var result [64]int16
	result[0] = int16(math.Round(float64(dct[0] / quantTable[0])))

	// Candidates follow mozjpeg's quantize_trellis (jcdctmgr.c): a coefficient
	// that rounds to 0 stays 0, otherwise each size category up to the rounded
	// magnitude qv offers only its largest magnitude (1, 3, 7, ..., then qv).
	// Within a category the rate is the same, so the largest magnitude has the
	// least distortion, and magnitudes above qv lose on rate and distortion.
	//
	// A nonzero coefficient at position i follows the previous nonzero one at
	// position j (or the DC at j = 0) after a run of i-1-j zeros. With
	// zeroSum[i] the summed zero distortion of positions 1..i, that state costs
	// base[j] + zeroSum[i-1] + rate + distortion, where base[j] is the best cost
	// ending nonzero at j minus zeroSum[j]. Only positions that can be nonzero
	// are predecessors, so a block costs about K² · log(qv) for K such positions.
	var (
		base         [64]float64
		zeroSums     [64]float64
		previous     [64]uint8 // Predecessor position of the best nonzero state.
		values       [64]int16
		predecessors [64]uint8 // Positions that can be nonzero, starting with DC.
	)
	count := 1
	rates := &acRates[0]
	if !isLuminance {
		rates = &acRates[1]
	}
	zrlRate := rates.byRun[0][15]
	eobRate := rates.byRun[0][0]
	for i := 1; i < 64; i++ {
		index := Zigzag[i]
		step := quantTable[index]
		zeroSums[i] = zeroSums[i-1] + config.Lambda*calculateDistortion(dct[index], 0, index, config.UsePerceptual)
		magnitude := math.Round(math.Abs(float64(dct[index] / step)))
		if magnitude == 0 {
			continue
		}
		qv := int(min(magnitude, 1023))
		sign := 1
		if dct[index] < 0 {
			sign = -1
		}

		best := math.Inf(1)
		for category := 1; qv>>(category-1) != 0; category++ {
			candidate := qv
			if qv>>category != 0 {
				candidate = 1<<category - 1
			}
			value := int16(sign * candidate)
			distortion := config.Lambda * calculateDistortion(dct[index], float32(value)*step, index, config.UsePerceptual)
			byRun := &rates.byRun[category]
			for p := 0; p < count; p++ {
				j := int(predecessors[p])
				run := i - 1 - j
				cost := base[j] + zeroSums[i-1] + float64((run>>4)*zrlRate+byRun[run&15]) + distortion
				if cost < best {
					best, values[i], previous[i] = cost, value, uint8(j)
				}
			}
		}
		base[i] = best - zeroSums[i]
		predecessors[count] = uint8(i)
		count++
	}

	// The block ends after its last nonzero coefficient; every earlier end
	// needs an EOB, and ending at DC means all AC coefficients are zero.
	last, bestCost := 0, zeroSums[63]+float64(eobRate)
	for p := 1; p < count; p++ {
		j := int(predecessors[p])
		cost := base[j] + zeroSums[63]
		if j < 63 {
			cost += float64(eobRate)
		}
		if cost < bestCost {
			last, bestCost = j, cost
		}
	}
	for j := last; j > 0; j = int(previous[j]) {
		result[Zigzag[j]] = values[j]
	}
	return result
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
// byRun[category][run].
type acRateTable struct {
	byRun [16][16]int
}

// acRates holds luminance ([0]) and chrominance ([1]) rates. The trellis inner
// loop runs O(64²) times per block, so table lookups replace Huffman calls.
var acRates = func() (tables [2]acRateTable) {
	for t, isLuminance := range []bool{true, false} {
		for category := 0; category < 16; category++ {
			for run := 0; run < 16; run++ {
				rate := estimateACRate(uint8(run), uint8(category), isLuminance)
				tables[t].byRun[category][run] = rate
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
