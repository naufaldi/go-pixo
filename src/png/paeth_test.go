package png

import "testing"

func TestPaethPredictor(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		c        int
		expected int
	}{
		{"a closest", 1, 2, 3, 1},
		{"b closest", 2, 1, 3, 1},
		{"c closest", 3, 2, 1, 3},
		{"a and b equal", 1, 1, 3, 1},
		{"all equal", 5, 5, 5, 5},
		{"p equals c", 10, 20, 15, 15},
		{"p equals c", 20, 10, 15, 15},
		{"b closest", 15, 20, 10, 20},
		{"negative values", -5, 5, 0, 0},
		{"large values", 255, 128, 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PaethPredictor(tt.a, tt.b, tt.c)
			if result != tt.expected {
				t.Errorf("PaethPredictor(%d, %d, %d) = %d, want %d",
					tt.a, tt.b, tt.c, result, tt.expected)
			}
		})
	}
}

func TestPaethPredictorPNGSpecExample(t *testing.T) {
	// PNG spec example: a=1, b=2, c=3
	// p = a + b - c = 1 + 2 - 3 = 0
	// pa = |0-1| = 1, pb = |0-2| = 2, pc = |0-3| = 3
	// pa <= pb && pa <= pc, so return a = 1
	result := PaethPredictor(1, 2, 3)
	if result != 1 {
		t.Errorf("PaethPredictor(1, 2, 3) = %d, want 1", result)
	}
}
