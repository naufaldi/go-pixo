package jpeg

import (
	"math"
	"testing"
)

func TestForwardDCT_Zeros(t *testing.T) {
	var block [64]float32
	result := ForwardDCT(block)
	for i, val := range result {
		if math.Abs(float64(val)) > 1e-6 {
			t.Errorf("at index %d: got %f, want 0", i, val)
		}
	}
}

func TestForwardDCT_Constant(t *testing.T) {
	var block [64]float32
	for i := range block {
		block[i] = 100.0
	}
	result := ForwardDCT(block)

	// DC component (index 0) should be 8 * 100 = 800
	if math.Abs(float64(result[0])-800.0) > 1e-3 {
		t.Errorf("DC component: got %f, want 800.0", result[0])
	}

	// AC components should be 0
	for i := 1; i < 64; i++ {
		if math.Abs(float64(result[i])) > 1e-3 {
			t.Errorf("AC component at %d: got %f, want 0", i, result[i])
		}
	}
}

func TestDCT_RoundTrip(t *testing.T) {
	var block [64]float32
	for i := range block {
		// Create some pattern
		block[i] = float32(i*4) - 128.0
	}

	dct := ForwardDCT(block)
	recovered := InverseDCT(dct)

	for i := range block {
		if math.Abs(float64(block[i]-recovered[i])) > 1e-3 {
			t.Errorf("at index %d: original %f, recovered %f, diff %f",
				i, block[i], recovered[i], math.Abs(float64(block[i]-recovered[i])))
		}
	}
}

func TestDCT_StepPattern(t *testing.T) {
	var block [64]float32
	// Left half 100, right half -100
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if col < 4 {
				block[row*8+col] = 100
			} else {
				block[row*8+col] = -100
			}
		}
	}

	result := ForwardDCT(block)

	// DC should be 0 (average is 0)
	if math.Abs(float64(result[0])) > 1e-3 {
		t.Errorf("DC component: got %f, want 0", result[0])
	}

	// Should have some non-zero AC components
	hasAC := false
	for i := 1; i < 64; i++ {
		if math.Abs(float64(result[i])) > 1.0 {
			hasAC = true
			break
		}
	}
	if !hasAC {
		t.Error("Expected some non-zero AC components for step pattern")
	}
}
