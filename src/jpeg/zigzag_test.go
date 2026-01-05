package jpeg

import (
	"testing"
)

func TestZigzagReorder(t *testing.T) {
	var block [64]int16
	for i := 0; i < 64; i++ {
		block[i] = int16(i)
	}

	reordered := ZigzagReorder(block)

	expected := []int16{
		0, 1, 8, 16, 9, 2, 3, 10,
		17, 24, 32, 25, 18, 11, 4, 5,
	}

	for i := 0; i < len(expected); i++ {
		if reordered[i] != expected[i] {
			t.Errorf("at index %d: got %d, want %d", i, reordered[i], expected[i])
		}
	}
}

func TestZigzag_RoundTrip(t *testing.T) {
	var block [64]int16
	for i := 0; i < 64; i++ {
		block[i] = int16(i)
	}

	reordered := ZigzagReorder(block)
	recovered := Dezigzag(reordered)

	for i := 0; i < 64; i++ {
		if recovered[i] != block[i] {
			t.Errorf("at index %d: recovered %d, original %d", i, recovered[i], block[i])
		}
	}
}

func TestZigzag_Completeness(t *testing.T) {
	seen := make(map[int]bool)
	for _, pos := range Zigzag {
		if seen[pos] {
			t.Errorf("duplicate position %d in zigzag", pos)
		}
		seen[pos] = true
	}
	for i := 0; i < 64; i++ {
		if !seen[i] {
			t.Errorf("position %d missing from zigzag", i)
		}
	}
}
