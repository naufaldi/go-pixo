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

	dcLuminanceBits   = [16]uint8{0, 1, 5, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0}
	dcLuminanceVals   = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	dcChrominanceBits = [16]uint8{0, 3, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0}
	dcChrominanceVals = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	acLuminanceBits   = [16]uint8{0, 2, 1, 3, 3, 2, 4, 3, 5, 5, 4, 4, 0, 0, 1, 125}
	acChrominanceBits = [16]uint8{0, 2, 1, 2, 4, 4, 3, 4, 7, 5, 4, 4, 0, 1, 2, 119}
)

type TrellisConfig struct {
	Lambda        float64
	MaxIterations int
	UsePerceptual bool
}

var DefaultTrellisConfig = TrellisConfig{
	Lambda:        1.0,
	MaxIterations: 64,
	UsePerceptual: true,
}

func PerceptualDistortion(dct, quantized [64]float32) float64 {
	totalDistortion := 0.0
	for i := 0; i < 64; i++ {
		diff := float64(dct[i] - quantized[i])
		weightedDistortion := diff * diff * visualSensitivityWeights[i]
		totalDistortion += weightedDistortion
	}
	return totalDistortion
}

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

type viterbiState struct {
	cost         float64
	quantizedVal int16
	prevIndex    int
	runLength    int
	isEOB        bool
}

type candidateValue struct {
	value      int16
	distortion float64
	rate       int
	cost       float64
}

func TrellisOptimizeFull(dct [64]float32, quantTable [64]float32, lambda float64) [64]int16 {
	config := DefaultTrellisConfig
	config.Lambda = lambda
	return TrellisOptimizeWithConfig(dct, quantTable, config)
}

func TrellisOptimizeWithConfig(dct [64]float32, quantTable [64]float32, config TrellisConfig) [64]int16 {
	if config.Lambda == 0 {
		config.Lambda = DefaultTrellisConfig.Lambda
	}
	if config.MaxIterations == 0 {
		config.MaxIterations = DefaultTrellisConfig.MaxIterations
	}

	result := [64]int16{}

	bestStates := make([][]viterbiState, 64)
	for i := 0; i < 64; i++ {
		bestStates[i] = make([]viterbiState, 0, 8)
	}

	candidates := generateCandidates(dct[0], quantTable[0], -255, 255)
	for _, cand := range candidates {
		distortion := calculateDistortion(dct[0], float32(cand)*quantTable[0], 0, config.UsePerceptual)
		rate := estimateDCRate(int16(cand), true)
		cost := float64(rate) + config.Lambda*distortion

		bestStates[0] = append(bestStates[0], viterbiState{
			cost:         cost,
			quantizedVal: int16(cand),
			prevIndex:    -1,
			runLength:    0,
			isEOB:        false,
		})
	}

	for i := 1; i < 64; i++ {
		step := quantTable[i]

		for runLength := 0; runLength <= 16; runLength++ {
			candidates := generateCandidatesWithRunLength(dct[i], step, runLength, -255, 255)

			for _, cand := range candidates {
				var distortion float64
				if config.UsePerceptual {
					quantizedVal := float32(cand.value) * step
					distortion = calculatePerceptualDistortion(dct[i], quantizedVal, i)
				} else {
					distortion = calculateSquaredDistortion(dct[i], float32(cand.value)*step)
				}

				var prevBest *viterbiState
				bestCost := math.Inf(1)

				for _, state := range bestStates[i-1] {
					newRunLength := state.runLength + 1
					if cand.value == 0 {
						newRunLength = state.runLength + 1
					} else {
						newRunLength = 0
					}

					var rate int
					if cand.value == 0 && newRunLength >= 16 {
						rate = estimateACRate(15, 0, true)
					} else if cand.value == 0 {
						rate = estimateACRate(uint8(newRunLength), 0, true)
					} else {
						acCategory := getACCategory(int16(cand.value))
						rate = estimateACRate(uint8(newRunLength), uint8(acCategory), true)
					}

					cost := state.cost + float64(rate) + config.Lambda*distortion

					if cost < bestCost {
						bestCost = cost
						prevState := state
						prevBest = &prevState
					}
				}

				if prevBest != nil {
					newState := viterbiState{
						cost:         bestCost,
						quantizedVal: int16(cand.value),
						prevIndex:    i - 1,
						runLength:    0,
						isEOB:        false,
					}
					if cand.value == 0 {
						newState.runLength = prevBest.runLength + 1
					}

					addStateWithPruning(bestStates[i], newState, 8)
				}
			}
		}

		if len(bestStates[i]) == 0 {
			candidates := generateCandidates(dct[i], step, -255, 255)
			for _, cand := range candidates {
				var distortion float64
				if config.UsePerceptual {
					quantizedVal := float32(cand) * step
					distortion = calculatePerceptualDistortion(dct[i], quantizedVal, i)
				} else {
					distortion = calculateSquaredDistortion(dct[i], float32(cand)*step)
				}
				rate := estimateACRate(0, uint8(getACCategory(int16(cand))), true)
				cost := float64(rate) + config.Lambda*distortion

				bestStates[i] = append(bestStates[i], viterbiState{
					cost:         cost,
					quantizedVal: int16(cand),
					prevIndex:    i - 1,
					runLength:    0,
					isEOB:        false,
				})
			}
		}
	}

	var bestFinalState *viterbiState
	bestFinalCost := math.Inf(1)
	for _, state := range bestStates[63] {
		var eobRate int
		if state.runLength > 0 {
			eobRate = estimateACRate(uint8(state.runLength), 0, true)
		} else {
			eobRate = 0
		}
		totalCost := state.cost + float64(eobRate)

		if totalCost < bestFinalCost {
			bestFinalCost = totalCost
			bestFinalState = &state
		}
	}

	if bestFinalState != nil {
		currentIndex := 63
		for currentIndex >= 0 {
			for _, state := range bestStates[currentIndex] {
				if state.cost == bestFinalState.cost && state.quantizedVal == bestFinalState.quantizedVal {
					result[currentIndex] = state.quantizedVal
					if state.prevIndex >= 0 {
						bestFinalState = &state
					}
					break
				}
			}
			currentIndex--
			if currentIndex < 0 {
				break
			}
			for _, s := range bestStates[currentIndex] {
				if s.quantizedVal == result[currentIndex+1] {
					bestFinalState = &s
					break
				}
			}
		}
	}

	return result
}

func generateCandidates(coeff, step float32, minVal, maxVal int) []int16 {
	centerVal := int(math.Round(float64(coeff / step)))

	candidates := make([]int16, 0)
	for delta := -3; delta <= 3; delta++ {
		candidate := centerVal + delta
		if candidate >= minVal && candidate <= maxVal {
			candidates = append(candidates, int16(candidate))
		}
	}

	return candidates
}

func generateCandidatesWithRunLength(coeff, step float32, runLength int, minVal, maxVal int) []candidateValue {
	candidates := make([]candidateValue, 0)

	if runLength >= 16 {
		candidates = append(candidates, candidateValue{value: 0, distortion: 0, rate: 0, cost: 0})
		return candidates
	}

	centerVal := int(math.Round(float64(coeff / step)))

	rangeSize := 3
	if coeff == 0 {
		rangeSize = 1
	}

	for delta := -rangeSize; delta <= rangeSize; delta++ {
		candidate := centerVal + delta
		if candidate < minVal || candidate > maxVal {
			continue
		}

		quantizedVal := float32(candidate) * step
		distortion := calculateSquaredDistortion(coeff, quantizedVal)
		rate := 0
		if candidate == 0 {
			rate = estimateACRate(uint8(runLength+1), 0, true)
		} else {
			acCategory := getACCategory(int16(candidate))
			rate = estimateACRate(uint8(runLength), uint8(acCategory), true)
		}
		cost := float64(rate) + distortion

		candidates = append(candidates, candidateValue{
			value:      int16(candidate),
			distortion: distortion,
			rate:       rate,
			cost:       cost,
		})
	}

	return candidates
}

func addStateWithPruning(states []viterbiState, newState viterbiState, maxStates int) {
	for i, s := range states {
		if s.quantizedVal == newState.quantizedVal && s.runLength == newState.runLength {
			if newState.cost < s.cost {
				states[i] = newState
			}
			return
		}
	}

	states = append(states, newState)

	for i := 0; i < len(states)-1; i++ {
		for j := i + 1; j < len(states); j++ {
			if states[i].cost > states[j].cost {
				states[i], states[j] = states[j], states[i]
			}
		}
	}

	if len(states) > maxStates {
		states = states[:maxStates]
	}
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
	absVal := int16(absInt16(quantVal))

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

func CalculateLambda(quality uint8) float32 {
	qualityFloat := float32(quality)
	if qualityFloat >= 50 {
		return 0.1 + (100-qualityFloat)/50*0.9
	} else {
		return 1.0 + (50-qualityFloat)/50*4.0
	}
}
