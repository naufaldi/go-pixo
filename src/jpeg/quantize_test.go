package jpeg

import (
	"testing"
)

func TestNewQuantizationTables(t *testing.T) {
	t.Run("Quality50", func(t *testing.T) {
		qt := NewQuantizationTables(50)
		// At quality 50, scale is 100, so tables should match standard tables
		for i := 0; i < 64; i++ {
			if qt.LuminanceTable[i] != stdLuminanceTable[i] {
				t.Errorf("Luminance[%d]: got %d, want %d", i, qt.LuminanceTable[i], stdLuminanceTable[i])
			}
			if qt.ChrominanceTable[i] != stdChrominanceTable[i] {
				t.Errorf("Chrominance[%d]: got %d, want %d", i, qt.ChrominanceTable[i], stdChrominanceTable[i])
			}
		}
	})

	t.Run("Quality100", func(t *testing.T) {
		qt := NewQuantizationTables(100)
		// At quality 100, all values should be 1
		for i := 0; i < 64; i++ {
			if qt.LuminanceTable[i] != 1 {
				t.Errorf("Luminance[%d]: got %d, want 1", i, qt.LuminanceTable[i])
			}
			if qt.ChrominanceTable[i] != 1 {
				t.Errorf("Chrominance[%d]: got %d, want 1", i, qt.ChrominanceTable[i])
			}
		}
	})

	t.Run("Quality1", func(t *testing.T) {
		qt := NewQuantizationTables(1)
		// At quality 1, all values should be large (capped at 255)
		if qt.LuminanceTable[0] != 255 {
			t.Errorf("Luminance[0]: got %d, want 255", qt.LuminanceTable[0])
		}
	})
}

func TestQuantizeBlock(t *testing.T) {
	var dct [64]float32
	dct[0] = 160.0
	dct[1] = 16.5
	dct[2] = -32.0

	var table [64]float32
	for i := range table {
		table[i] = 16.0
	}

	result := QuantizeBlock(dct, table)

	if result[0] != 10 {
		t.Errorf("result[0]: got %d, want 10", result[0])
	}
	if result[1] != 1 {
		t.Errorf("result[1]: got %d, want 1", result[1])
	}
	if result[2] != -2 {
		t.Errorf("result[2]: got %d, want -2", result[2])
	}
}
