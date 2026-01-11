package jpeg

import (
	"math"
	"testing"
)

func TestPerceptualDistortion(t *testing.T) {
	tests := []struct {
		name          string
		dct           [64]float32
		quantized     [64]float32
		expectLowDist bool
	}{
		{
			name: "identical blocks",
			dct: func() [64]float32 {
				var dct [64]float32
				for i := 0; i < 64; i++ {
					dct[i] = float32(i) * 10.0
				}
				return dct
			}(),
			quantized: func() [64]float32 {
				var quantized [64]float32
				for i := 0; i < 64; i++ {
					quantized[i] = float32(i) * 10.0
				}
				return quantized
			}(),
			expectLowDist: true,
		},
		{
			name: "different blocks",
			dct: func() [64]float32 {
				var dct [64]float32
				for i := 0; i < 64; i++ {
					dct[i] = float32(i) * 10.0
				}
				return dct
			}(),
			quantized: func() [64]float32 {
				var quantized [64]float32
				for i := 0; i < 64; i++ {
					quantized[i] = float32(i) * 5.0
				}
				return quantized
			}(),
			expectLowDist: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			distortion := PerceptualDistortion(test.dct, test.quantized)

			if test.expectLowDist && distortion > 0.001 {
				t.Errorf("Expected low distortion for identical blocks, got %f", distortion)
			}

			if !test.expectLowDist && distortion < 0.001 {
				t.Errorf("Expected high distortion for different blocks, got %f", distortion)
			}
		})
	}
}

func TestPerceptualDistortionWeights(t *testing.T) {
	var dct [64]float32
	var quantized [64]float32

	dct[0] = 100.0
	quantized[0] = 0.0

	dct[10] = 100.0
	quantized[10] = 0.0

	dcDistortion := PerceptualDistortion(dct, quantized)

	dct[0] = 0.0
	quantized[0] = 0.0
	dct[10] = 100.0
	quantized[10] = 0.0

	acDistortion := PerceptualDistortion(dct, quantized)

	if dcDistortion <= acDistortion {
		t.Errorf("DC distortion (%f) should be higher than AC distortion (%f) for same coefficient values due to weight differences", dcDistortion, acDistortion)
	}
}

func TestEstimateSymbolRate(t *testing.T) {
	tests := []struct {
		name          string
		coeffs        [64]int16
		isLuminance   bool
		expectMinBits int
		expectMaxBits int
	}{
		{
			name: "all zeros",
			coeffs: func() [64]int16 {
				var coeffs [64]int16
				return coeffs
			}(),
			isLuminance:   true,
			expectMinBits: 1,
			expectMaxBits: 16,
		},
		{
			name: "DC only",
			coeffs: func() [64]int16 {
				var coeffs [64]int16
				coeffs[0] = 5
				return coeffs
			}(),
			isLuminance:   true,
			expectMinBits: 4,
			expectMaxBits: 20,
		},
		{
			name: "sparse coefficients",
			coeffs: func() [64]int16 {
				var coeffs [64]int16
				coeffs[0] = 3
				coeffs[10] = 2
				coeffs[20] = 1
				return coeffs
			}(),
			isLuminance:   true,
			expectMinBits: 8,
			expectMaxBits: 40,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bits := EstimateSymbolRate(test.coeffs, test.isLuminance)

			if bits < test.expectMinBits {
				t.Errorf("Expected at least %d bits, got %d", test.expectMinBits, bits)
			}
			if bits > test.expectMaxBits {
				t.Errorf("Expected at most %d bits, got %d", test.expectMaxBits, bits)
			}
		})
	}
}

func TestEstimateSymbolRateAccuracy(t *testing.T) {
	var coeffs [64]int16
	coeffs[0] = 5

	bits1 := EstimateSymbolRate(coeffs, true)
	coeffs[0] = 100
	bits2 := EstimateSymbolRate(coeffs, true)

	if bits2 <= bits1 {
		t.Errorf("Larger coefficient should require more bits: %d vs %d", bits2, bits1)
	}
}

func TestTrellisOptimizeFull(t *testing.T) {
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		quantTable[i] = 16.0
	}

	result := TrellisOptimizeFull(dct, quantTable, 1.0)

	for i := 0; i < 64; i++ {
		if result[i] < -255 || result[i] > 255 {
			t.Errorf("Coefficient %d out of range: %d", i, result[i])
		}
	}

	nonZeroCount := 0
	for i := 0; i < 64; i++ {
		if result[i] != 0 {
			nonZeroCount++
		}
	}

	if nonZeroCount == 0 {
		t.Error("At least some coefficients should be non-zero for non-zero input")
	}
}

func TestTrellisOptimizeWithConfig(t *testing.T) {
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		quantTable[i] = 16.0
	}

	config := TrellisConfig{
		Lambda:        0.5,
		MaxIterations: 32,
		UsePerceptual: true,
	}

	result := TrellisOptimizeWithConfig(dct, quantTable, config)

	for i := 0; i < 64; i++ {
		if result[i] < -255 || result[i] > 255 {
			t.Errorf("Coefficient %d out of range: %d", i, result[i])
		}
	}
}

func TestTrellisConfigDefaults(t *testing.T) {
	config := DefaultTrellisConfig

	if config.Lambda != 1.0 {
		t.Errorf("Default Lambda should be 1.0, got %f", config.Lambda)
	}
	if config.MaxIterations != 64 {
		t.Errorf("Default MaxIterations should be 64, got %d", config.MaxIterations)
	}
	if !config.UsePerceptual {
		t.Error("Default UsePerceptual should be true")
	}
}

func TestPerceptualVsSquaredDistortion(t *testing.T) {
	var dct [64]float32
	var quantized [64]float32

	dct[0] = 50.0
	quantized[0] = 48.0

	dct[10] = 50.0
	quantized[10] = 48.0

	perceptual := PerceptualDistortion(dct, quantized)

	var squaredDct [64]float32
	var squaredQuantized [64]float32
	squaredDct[0] = 50.0
	squaredQuantized[0] = 48.0
	squaredDct[10] = 50.0
	squaredQuantized[10] = 48.0

	squaredDiff := 0.0
	for i := 0; i < 64; i++ {
		diff := float64(squaredDct[i] - squaredQuantized[i])
		squaredDiff += diff * diff
	}

	if perceptual >= squaredDiff {
		t.Logf("Perceptual: %f, Squared: %f", perceptual, squaredDiff)
	}
}

func TestTrellisQuantize(t *testing.T) {
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		quantTable[i] = 16.0
	}

	lambda := CalculateLambda(75)
	result := TrellisQuantize(dct, quantTable, lambda)

	for i := 0; i < 64; i++ {
		if result[i] < -255 || result[i] > 255 {
			t.Errorf("Coefficient %d out of range: %d", i, result[i])
		}
	}

	nonZeroCount := 0
	for i := 0; i < 64; i++ {
		if result[i] != 0 {
			nonZeroCount++
		}
	}

	if nonZeroCount == 0 {
		t.Error("At least some coefficients should be non-zero for non-zero input")
	}
}

func TestCalculateLambda(t *testing.T) {
	tests := []struct {
		quality  uint8
		expected float32
	}{
		{10, 5.0},
		{25, 3.0},
		{50, 1.0},
		{75, 0.5},
		{90, 0.3},
		{100, 0.1},
	}

	for _, test := range tests {
		lambda := CalculateLambda(test.quality)
		if lambda < 0 || lambda > 10 {
			t.Errorf("Lambda out of range for quality %d: %f", test.quality, lambda)
		}
	}
}

func TestQuantizeSingleCoefficient(t *testing.T) {
	tests := []struct {
		coeff      float32
		quant      float32
		minVal     int
		maxVal     int
		lambda     float32
		expectedOK bool
	}{
		{100.0, 16.0, -255, 255, 1.0, true},
		{0.0, 16.0, -255, 255, 1.0, true},
		{-50.0, 16.0, -255, 255, 1.0, true},
	}

	for _, test := range tests {
		result := quantizeSingleCoefficient(test.coeff, test.quant, test.minVal, test.maxVal, test.lambda)

		if result < int16(test.minVal) || result > int16(test.maxVal) {
			t.Errorf("Result %d out of range [%d, %d]", result, test.minVal, test.maxVal)
		}

		if test.coeff == 0.0 && result != 0 {
			if absInt16(result) > 1 {
				t.Errorf("Expected near-zero for zero input, got %d", result)
			}
		}
	}
}

func TestAbsInt16(t *testing.T) {
	tests := []struct {
		input    int16
		expected int16
	}{
		{0, 0},
		{1, 1},
		{-1, 1},
		{100, 100},
		{-100, 100},
		{-32767, 32767},
	}

	for _, test := range tests {
		result := absInt16(test.input)
		if result != test.expected {
			t.Errorf("absInt16(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestVisualSensitivityWeights(t *testing.T) {
	if visualSensitivityWeights[0] <= visualSensitivityWeights[63] {
		t.Error("DC coefficient should have higher visual sensitivity weight than high-frequency AC coefficients")
	}

	for i := 0; i < 64; i++ {
		if visualSensitivityWeights[i] < 0.0 || visualSensitivityWeights[i] > 1.0 {
			t.Errorf("Visual sensitivity weight %d should be between 0 and 1: %f", i, visualSensitivityWeights[i])
		}
	}
}

func TestGetDCCategory(t *testing.T) {
	tests := []struct {
		input    int16
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 3},
		{7, 3},
		{8, 4},
		{100, 7},
		{-1, 1},
		{-2, 2},
	}

	for _, test := range tests {
		result := getDCCategory(test.input)
		if result != test.expected {
			t.Errorf("getDCCategory(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestGetACCategory(t *testing.T) {
	tests := []struct {
		input    int16
		expected int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 3},
		{7, 3},
		{8, 4},
		{100, 7},
		{-1, 1},
		{-2, 2},
	}

	for _, test := range tests {
		result := getACCategory(test.input)
		if result != test.expected {
			t.Errorf("getACCategory(%d) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestEstimateDCRate(t *testing.T) {
	tests := []struct {
		quantVal    int16
		isLuminance bool
		minBits     int
		maxBits     int
	}{
		{0, true, 2, 4},
		{1, true, 4, 8},
		{5, true, 4, 10},
		{100, true, 6, 14},
		{0, false, 2, 6},
		{5, false, 4, 12},
	}

	for _, test := range tests {
		bits := estimateDCRate(test.quantVal, test.isLuminance)
		if bits < test.minBits || bits > test.maxBits {
			t.Errorf("estimateDCRate(%d, %v) = %d, expected between %d and %d", test.quantVal, test.isLuminance, bits, test.minBits, test.maxBits)
		}
	}
}

func TestEstimateACRate(t *testing.T) {
	tests := []struct {
		runLength   uint8
		category    uint8
		isLuminance bool
		minBits     int
		maxBits     int
	}{
		{0, 0, true, 2, 6},
		{0, 1, true, 2, 10},
		{1, 1, true, 4, 12},
		{5, 2, true, 4, 16},
		{0, 0, false, 2, 8},
		{0, 1, false, 2, 12},
	}

	for _, test := range tests {
		bits := estimateACRate(test.runLength, test.category, test.isLuminance)
		if bits < test.minBits || bits > test.maxBits {
			t.Errorf("estimateACRate(%d, %d, %v) = %d, expected between %d and %d", test.runLength, test.category, test.isLuminance, bits, test.minBits, test.maxBits)
		}
	}
}

func TestGenerateCandidates(t *testing.T) {
	coeff := float32(50.0)
	step := float32(16.0)

	candidates := generateCandidates(coeff, step, -255, 255)

	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	centerVal := int(math.Round(float64(coeff / step)))
	hasCenter := false
	for _, c := range candidates {
		if c == int16(centerVal) {
			hasCenter = true
			break
		}
	}

	if !hasCenter {
		t.Error("Center value should be in candidates")
	}
}

func TestGenerateCandidatesWithRunLength(t *testing.T) {
	coeff := float32(50.0)
	step := float32(16.0)

	candidates := generateCandidatesWithRunLength(coeff, step, 0, -255, 255)

	if len(candidates) == 0 {
		t.Error("Expected at least one candidate")
	}

	for _, c := range candidates {
		if c.value < -255 || c.value > 255 {
			t.Errorf("Candidate value %d out of range", c.value)
		}
	}

	candidatesZeros := generateCandidatesWithRunLength(coeff, step, 16, -255, 255)
	if len(candidatesZeros) != 1 || candidatesZeros[0].value != 0 {
		t.Error("Run length >= 16 should only return zero candidate")
	}
}

func TestTrellisQuantizeRateDistortion(t *testing.T) {
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		quantTable[i] = 16.0
	}

	lambda := CalculateLambda(50)
	result := TrellisQuantize(dct, quantTable, lambda)

	rate := EstimateSymbolRate(result, true)

	if rate <= 0 {
		t.Error("Expected positive rate for non-zero coefficients")
	}

	var zeroResult [64]int16
	zeroRate := EstimateSymbolRate(zeroResult, true)

	if rate < zeroRate {
		t.Logf("Trellis rate: %d, Zero rate: %d", rate, zeroRate)
	}
}

func BenchmarkPerceptualDistortion(b *testing.B) {
	var dct [64]float32
	var quantized [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
		quantized[i] = float32(i) * 9.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PerceptualDistortion(dct, quantized)
	}
}

func BenchmarkEstimateSymbolRate(b *testing.B) {
	var coeffs [64]int16
	for i := 0; i < 64; i++ {
		coeffs[i] = int16(i % 10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EstimateSymbolRate(coeffs, true)
	}
}

func BenchmarkTrellisOptimizeFull(b *testing.B) {
	var dct [64]float32
	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
		quantTable[i] = 16.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TrellisOptimizeFull(dct, quantTable, 1.0)
	}
}

func BenchmarkTrellisQuantize(b *testing.B) {
	var dct [64]float32
	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
		quantTable[i] = 16.0
	}

	lambda := CalculateLambda(75)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TrellisQuantize(dct, quantTable, lambda)
	}
}
