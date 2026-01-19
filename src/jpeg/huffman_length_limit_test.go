package jpeg

import "testing"

func TestHuffmanLengthLimit_RespectsMaxLen(t *testing.T) {
	freq := make([]int, 256)
	freq[0] = 1000000
	for i := 1; i < len(freq); i++ {
		freq[i] = 1
	}

	bits, vals := buildLengthLimitedHuffmanTable(freq, 16)

	if len(vals) == 0 {
		t.Fatalf("expected non-empty vals")
	}

	total := 0
	for i := 0; i < 16; i++ {
		total += int(bits[i])
	}
	if total != len(vals) {
		t.Fatalf("sum(bits)=%d does not match vals len=%d", total, len(vals))
	}

	lengths := expandLengths(bits, vals)
	maxLen := 0
	for _, l := range lengths {
		if l > maxLen {
			maxLen = l
		}
		if l < 1 || l > 16 {
			t.Fatalf("invalid code length %d", l)
		}
	}
	if maxLen > 16 {
		t.Fatalf("expected max length <= 16, got %d", maxLen)
	}
}

func TestHuffmanLengthLimit_CanonicalOrder(t *testing.T) {
	freq := make([]int, 64)
	for i := range freq {
		freq[i] = (i % 7) + 1
	}
	freq[3] = 500
	freq[17] = 300

	bits, vals := buildLengthLimitedHuffmanTable(freq, 16)
	lengths := expandLengths(bits, vals)

	for i := 1; i < len(vals); i++ {
		prevLen := lengths[i-1]
		currLen := lengths[i]
		if currLen < prevLen {
			t.Fatalf("lengths not non-decreasing at %d: %d -> %d", i, prevLen, currLen)
		}
		if currLen == prevLen && vals[i] < vals[i-1] {
			t.Fatalf("vals not ordered within length %d: %d -> %d", currLen, vals[i-1], vals[i])
		}
	}
}

func TestHuffmanLengthLimit_EmptyAndSingleSymbol(t *testing.T) {
	bits, vals := buildLengthLimitedHuffmanTable([]int{0, 0, 0}, 16)
	if len(vals) != 1 || vals[0] != 0 || bits[0] != 1 {
		t.Fatalf("empty case expected bits[0]=1 vals[0]=0, got bits[0]=%d vals=%v", bits[0], vals)
	}

	freq := []int{0, 0, 5, 0}
	bits, vals = buildLengthLimitedHuffmanTable(freq, 16)
	if len(vals) != 1 || vals[0] != 2 || bits[0] != 1 {
		t.Fatalf("single case expected bits[0]=1 vals[0]=2, got bits[0]=%d vals=%v", bits[0], vals)
	}
}

func expandLengths(bits [16]uint8, vals []uint8) []int {
	lengths := make([]int, len(vals))
	idx := 0
	for length := 1; length <= 16; length++ {
		count := int(bits[length-1])
		for i := 0; i < count; i++ {
			if idx >= len(vals) {
				return lengths
			}
			lengths[idx] = length
			idx++
		}
	}
	return lengths
}
