package jpeg

import (
	"testing"
)

func TestRunLengthEncode(t *testing.T) {
	t.Run("Zeros", func(t *testing.T) {
		var block [64]int16
		runs := RunLengthEncode(block)
		// Should only have EOB
		if len(runs) != 1 || runs[0].RunLength != 0 || runs[0].Size != 0 {
			t.Errorf("expected only EOB, got %+v", runs)
		}
	})

	t.Run("SingleValue", func(t *testing.T) {
		var block [64]int16
		block[1] = 5 // First AC coefficient
		runs := RunLengthEncode(block)
		// Should have (0, 3) for 5, then EOB
		if len(runs) != 2 {
			t.Errorf("expected 2 runs, got %d", len(runs))
		}
		if runs[0].RunLength != 0 || runs[0].Size != 3 || runs[0].Value != 5 {
			t.Errorf("first run: got %+v, want {0, 3, 5}", runs[0])
		}
		if runs[1].RunLength != 0 || runs[1].Size != 0 {
			t.Errorf("second run should be EOB, got %+v", runs[1])
		}
	})

	t.Run("LongZeroRun", func(t *testing.T) {
		var block [64]int16
		block[20] = 5 // 19 zeros before this
		runs := RunLengthEncode(block)
		// Should have ZRL (15 zeros), then (3, 3) for 5, then EOB
		// Wait, 19 zeros:
		// 15 zeros -> ZRL (15, 0)
		// 4 zeros remaining -> (4, 3) for value 5
		// No, let's re-calculate:
		// Index 1-19 are zero (19 coefficients).
		// (15, 0) handles index 1-16 (16 zeros? no, ZRL is 16 zeros).
		// If index 1-16 are zero, that's 16 zeros. ZRL.
		// Then index 17, 18, 19 are zero (3 zeros).
		// So (3, 3) for index 20.
		// Let's check my code:
		// i=1..19: zeroRun increments to 19.
		// i=20: val=5.
		//   while zeroRun >= 16:
		//     ZRL (15, 0), zeroRun = 3.
		//   runs += (3, 3, 5)
		// Correct.

		if len(runs) != 3 {
			t.Errorf("expected 3 runs, got %d", len(runs))
		}
		if runs[0].RunLength != 15 || runs[0].Size != 0 {
			t.Errorf("first run should be ZRL, got %+v", runs[0])
		}
		if runs[1].RunLength != 3 || runs[1].Size != 3 || runs[1].Value != 5 {
			t.Errorf("second run: got %+v, want {3, 3, 5}", runs[1])
		}
	})
}

func TestAC_RoundTrip(t *testing.T) {
	var block [64]int16
	block[1] = 10
	block[5] = -5
	block[20] = 3
	block[63] = 1

	runs := RunLengthEncode(block)
	recovered := RunLengthDecode(runs)

	for i := 1; i < 64; i++ {
		if recovered[i] != block[i] {
			t.Errorf("at index %d: recovered %d, original %d", i, recovered[i], block[i])
		}
	}
}
