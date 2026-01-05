package jpeg

// ACRun represents an AC coefficient run-length entry.
type ACRun struct {
	RunLength uint8 // 0-15
	Size      uint8 // Category 1-10
	Value     int16 // The actual coefficient value
}

// RunLengthEncode performs run-length encoding on an 8x8 block of quantized coefficients.
// The input block MUST be in zigzag scan order.
// It returns a slice of ACRun structs and handles EOB and ZRL markers.
func RunLengthEncode(coeffs [64]int16) []ACRun {
	var runs []ACRun
	zeroRun := uint8(0)

	// Skip DC coefficient (index 0)
	for i := 1; i < 64; i++ {
		val := coeffs[i]
		if val == 0 {
			zeroRun++
		} else {
			// Handle zero runs longer than 15
			for zeroRun >= 16 {
				// ZRL (Zero Run Length) is represented as (15, 0)
				runs = append(runs, ACRun{RunLength: 15, Size: 0, Value: 0})
				zeroRun -= 16
			}

			// Add (Run, Size) pair for non-zero coefficient
			cat := Category(val)
			runs = append(runs, ACRun{RunLength: zeroRun, Size: cat, Value: val})
			zeroRun = 0
		}
	}

	// Add EOB (End of Block) if there are trailing zeros
	if zeroRun > 0 {
		runs = append(runs, ACRun{RunLength: 0, Size: 0, Value: 0})
	}

	return runs
}

// RunLengthDecode is a helper for testing that reconstructs the AC part of a block.
func RunLengthDecode(runs []ACRun) [64]int16 {
	var coeffs [64]int16
	pos := 1 // Skip DC

	for _, run := range runs {
		if pos >= 64 {
			break
		}

		if run.RunLength == 0 && run.Size == 0 {
			// EOB: Remaining coefficients are zero
			break
		}

		if run.RunLength == 15 && run.Size == 0 {
			// ZRL: 16 zeros
			pos += 16
			continue
		}

		// (Run, Size) pair
		pos += int(run.RunLength)
		if pos < 64 {
			coeffs[pos] = run.Value
			pos++
		}
	}

	return coeffs
}
