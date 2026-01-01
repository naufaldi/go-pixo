package compress

import (
	"testing"
)

func TestCanonicalizeDeterministic(t *testing.T) {
	codes := map[int]Code{
		0: {Length: 2},
		1: {Length: 2},
		2: {Length: 3},
	}

	result1, lengths1 := Canonicalize(codes)
	result2, lengths2 := Canonicalize(codes)

	if len(result1) != len(result2) {
		t.Fatalf("Different result lengths: %d vs %d", len(result1), len(result2))
	}

	for i := 0; i < len(result1); i++ {
		if result1[i].Length != result2[i].Length {
			t.Errorf("Result[%d].Length mismatch: %d vs %d", i, result1[i].Length, result2[i].Length)
		}
		if result1[i].Bits != result2[i].Bits {
			t.Errorf("Result[%d].Bits mismatch: 0x%04X vs 0x%04X", i, result1[i].Bits, result2[i].Bits)
		}
	}

	if len(lengths1) != len(lengths2) {
		t.Errorf("Lengths slice mismatch: %d vs %d", len(lengths1), len(lengths2))
	}
}

func TestCanonicalizePrefixFree(t *testing.T) {
	codes := map[int]Code{
		0: {Length: 2},
		1: {Length: 2},
		2: {Length: 3},
		3: {Length: 3},
	}

	result, _ := Canonicalize(codes)

	for i := 0; i < len(result); i++ {
		if result[i].Length == 0 {
			continue
		}
		for j := i + 1; j < len(result); j++ {
			if result[j].Length == 0 {
				continue
			}
			codeI := result[i]
			codeJ := result[j]

			msbCodeI := ReverseBits(codeI.Bits, codeI.Length)
			msbCodeJ := ReverseBits(codeJ.Bits, codeJ.Length)

			minLen := codeI.Length
			if codeJ.Length < minLen {
				minLen = codeJ.Length
			}

			mask := uint16((1 << uint(minLen)) - 1)
			var prefixI, prefixJ uint16
			if codeI.Length >= minLen {
				prefixI = (msbCodeI >> uint(codeI.Length-minLen)) & mask
			} else {
				prefixI = msbCodeI & mask
			}
			if codeJ.Length >= minLen {
				prefixJ = (msbCodeJ >> uint(codeJ.Length-minLen)) & mask
			} else {
				prefixJ = msbCodeJ & mask
			}
			if prefixI == prefixJ {
				t.Errorf("Codes for symbols %d and %d share prefix: 0x%04X and 0x%04X (prefixes: 0x%04X and 0x%04X)",
					i, j, msbCodeI, msbCodeJ, prefixI, prefixJ)
			}
		}
	}
}

func TestCanonicalizeLSBFirst(t *testing.T) {
	codes := map[int]Code{
		0: {Length: 2},
		1: {Length: 2},
	}

	result, _ := Canonicalize(codes)

	if len(result) < 2 {
		t.Fatalf("Expected at least 2 codes (0,1), got %d", len(result))
	}

	for symbol := 0; symbol < len(result); symbol++ {
		code := result[symbol]
		if symbol < 2 && code.Length == 0 {
			t.Errorf("Code for symbol %d has zero length", symbol)
		}
		if code.Length > 0 {
			msbCode := ReverseBits(code.Bits, code.Length)
			if msbCode >= (1 << uint(code.Length)) {
				t.Errorf("Code for symbol %d with length %d has MSB code %d >= 2^%d", symbol, code.Length, msbCode, code.Length)
			}
		}
	}
}

func TestGenerateCodes(t *testing.T) {
	frequencies := []int{3, 2, 1, 1}
	tree := BuildTree(frequencies)
	if tree == nil {
		t.Fatal("BuildTree returned nil")
	}

	codes := GenerateCodes(tree)
	if len(codes) == 0 {
		t.Fatal("GenerateCodes returned empty map")
	}

	for symbol, code := range codes {
		if code.Length == 0 {
			t.Errorf("Symbol %d has zero code length", symbol)
		}
	}
}

func TestBuildTreeAndCanonicalize(t *testing.T) {
	frequencies := []int{5, 3, 2, 1}
	tree := BuildTree(frequencies)
	if tree == nil {
		t.Fatal("BuildTree returned nil")
	}

	codes := GenerateCodes(tree)
	if len(codes) == 0 {
		t.Fatal("GenerateCodes returned empty map")
	}

	canonical, lengths := Canonicalize(codes)
	if len(canonical) == 0 {
		t.Fatal("Canonicalize returned empty result")
	}

	for symbol := 0; symbol < len(canonical); symbol++ {
		code := canonical[symbol]
		if symbol < len(lengths) && code.Length != lengths[symbol] {
			t.Errorf("Code[%d].Length = %d, but lengths[%d] = %d", symbol, code.Length, symbol, lengths[symbol])
		}
		if code.Length > 0 {
			msbCode := ReverseBits(code.Bits, code.Length)
			if msbCode == 0 && code.Length == 1 {
				t.Logf("Code[%d] has length %d with MSB code 0 (this is valid for length-1 code)", symbol, code.Length)
			} else if msbCode == 0 && code.Length > 1 {
				t.Errorf("Code[%d] has length %d but zero MSB bits", symbol, code.Length)
			}
		}
	}
}
