package compress

import (
	"testing"
)

func TestLiteralLengthTable_RFC1951Compliance(t *testing.T) {
	table := LiteralLengthTable()

	if len(table.Codes) != 288 {
		t.Fatalf("Expected 288 codes, got %d", len(table.Codes))
	}

	if table.MaxLength != 9 {
		t.Errorf("Expected MaxLength 9, got %d", table.MaxLength)
	}

	expectedLengths := make([]int, 288)
	for i := 0; i < 144; i++ {
		expectedLengths[i] = 8
	}
	for i := 144; i < 256; i++ {
		expectedLengths[i] = 9
	}
	for i := 256; i < 280; i++ {
		expectedLengths[i] = 7
	}
	for i := 280; i < 288; i++ {
		expectedLengths[i] = 8
	}

	for symbol := 0; symbol < 288; symbol++ {
		code := table.Codes[symbol]
		expectedLength := expectedLengths[symbol]

		if code.Length != expectedLength {
			t.Errorf("Symbol %d: expected length %d, got %d", symbol, expectedLength, code.Length)
		}

		if code.Length > 0 {
			maxBits := uint16((1 << uint(code.Length)) - 1)
			if code.Bits > maxBits {
				t.Errorf("Symbol %d: bits 0x%04X exceeds maximum 0x%04X for length %d", symbol, code.Bits, maxBits, code.Length)
			}
		}
	}
}

func TestDistanceTable_RFC1951Compliance(t *testing.T) {
	table := DistanceTable()

	if len(table.Codes) != 30 {
		t.Fatalf("Expected 30 codes, got %d", len(table.Codes))
	}

	if table.MaxLength != 5 {
		t.Errorf("Expected MaxLength 5, got %d", table.MaxLength)
	}

	for symbol := 0; symbol < 30; symbol++ {
		code := table.Codes[symbol]

		if code.Length != 5 {
			t.Errorf("Symbol %d: expected length 5, got %d", symbol, code.Length)
		}

		maxBits := uint16((1 << 5) - 1)
		if code.Bits > maxBits {
			t.Errorf("Symbol %d: bits 0x%04X exceeds maximum 0x%04X for length 5", symbol, code.Bits, maxBits)
		}
	}
}

func TestFixedTables_PrefixFree(t *testing.T) {
	litTable := LiteralLengthTable()
	distTable := DistanceTable()

	tables := []struct {
		name  string
		table Table
		max   int
	}{
		{"LiteralLength", litTable, 288},
		{"Distance", distTable, 30},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.max; i++ {
				codeI := tt.table.Codes[i]
				if codeI.Length == 0 {
					continue
				}

				for j := i + 1; j < tt.max; j++ {
					codeJ := tt.table.Codes[j]
					if codeJ.Length == 0 {
						continue
					}

					if isPrefix(codeI, codeJ) {
						t.Errorf("Codes for symbols %d and %d share prefix: %d bits (0x%04X) and %d bits (0x%04X)",
							i, j, codeI.Length, codeI.Bits, codeJ.Length, codeJ.Bits)
					}
				}
			}
		})
	}
}

func TestFixedTables_Structure(t *testing.T) {
	litTable := LiteralLengthTable()
	distTable := DistanceTable()

	if len(litTable.Codes) != 288 {
		t.Errorf("LiteralLengthTable: expected 288 codes, got %d", len(litTable.Codes))
	}

	if len(distTable.Codes) != 30 {
		t.Errorf("DistanceTable: expected 30 codes, got %d", len(distTable.Codes))
	}

	if litTable.MaxLength < 7 || litTable.MaxLength > 9 {
		t.Errorf("LiteralLengthTable: MaxLength should be 7-9, got %d", litTable.MaxLength)
	}

	if distTable.MaxLength != 5 {
		t.Errorf("DistanceTable: MaxLength should be 5, got %d", distTable.MaxLength)
	}
}

func isPrefix(code1, code2 Code) bool {
	minLen := code1.Length
	if code2.Length < minLen {
		minLen = code2.Length
	}

	mask := uint16((1 << uint(minLen)) - 1)
	bits1 := code1.Bits & mask
	bits2 := code2.Bits & mask

	return bits1 == bits2
}
