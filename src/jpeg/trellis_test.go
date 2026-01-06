package jpeg

import (
	"testing"
)

func TestTrellisQuantize(t *testing.T) {
	// Create a test DCT block
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	// Create a standard quantization table
	var quantTable [64]float32
	for i := 0; i < 64; {
		quantTable[i] = 16 // DC coefficient
		i++
		for j := 0; j < 15 && i < 64; j++ {
			quantTable[i] = 15 // AC coefficients
			i++
		}
	}

	// Test trellis quantization
	lambda := CalculateLambda(75)
	result := TrellisQuantize(dct, quantTable, lambda)

	// Verify all coefficients are within valid range
	for i := 0; i < 64; i++ {
		if result[i] < -255 || result[i] > 255 {
			t.Errorf("Coefficient %d out of range: %d", i, result[i])
		}
	}

	// Verify at least some coefficients are non-zero (we have non-zero input)
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
		expected float32 // approximate range
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

		// For zero input, expect zero or near-zero output
		if test.coeff == 0.0 && result != 0 {
			// It's OK to have small non-zero values due to the algorithm
			// Just verify it's close to zero
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

func BenchmarkTrellisQuantize(b *testing.B) {
	// Create test data
	var dct [64]float32
	for i := 0; i < 64; i++ {
		dct[i] = float32(i) * 10.0
	}

	var quantTable [64]float32
	for i := 0; i < 64; i++ {
		quantTable[i] = 16.0
	}

	lambda := CalculateLambda(75)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = TrellisQuantize(dct, quantTable, lambda)
	}
}
