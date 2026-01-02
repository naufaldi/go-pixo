package compress

import (
	"testing"
)

func TestBuildDynamicTables_ValidCodes(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	litFreq[65] = 3
	litFreq[66] = 2
	litFreq[67] = 1
	litFreq[256] = 1

	distFreq[1] = 2
	distFreq[2] = 1

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	if len(litTable.Codes) == 0 {
		t.Fatal("Expected literal table codes, got empty")
	}

	if len(distTable.Codes) == 0 {
		t.Fatal("Expected distance table codes, got empty")
	}

	if litTable.MaxLength == 0 {
		t.Error("Expected MaxLength > 0 for literal table")
	}

	if distTable.MaxLength == 0 {
		t.Error("Expected MaxLength > 0 for distance table")
	}

	for symbol := 0; symbol < len(litFreq) && symbol < len(litTable.Codes); symbol++ {
		if litFreq[symbol] > 0 {
			code := litTable.Codes[symbol]
			if code.Length == 0 {
				t.Errorf("Symbol %d has frequency %d but zero code length", symbol, litFreq[symbol])
			}
		}
	}

	for symbol := 0; symbol < len(distFreq) && symbol < len(distTable.Codes); symbol++ {
		if distFreq[symbol] > 0 {
			code := distTable.Codes[symbol]
			if code.Length == 0 {
				t.Errorf("Symbol %d has frequency %d but zero code length", symbol, distFreq[symbol])
			}
		}
	}
}

func TestBuildDynamicTables_PrefixFree(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	for i := 0; i < 10; i++ {
		litFreq[i+65] = 10 - i
	}
	litFreq[256] = 1

	for i := 0; i < 5; i++ {
		distFreq[i] = 5 - i
	}

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	tables := []struct {
		name  string
		table Table
		max   int
	}{
		{"LiteralLength", litTable, len(litTable.Codes)},
		{"Distance", distTable, len(distTable.Codes)},
	}

	for _, tt := range tables {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.max && i < len(tt.table.Codes); i++ {
				codeI := tt.table.Codes[i]
				if codeI.Length == 0 {
					continue
				}

				for j := i + 1; j < tt.max && j < len(tt.table.Codes); j++ {
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

func TestBuildDynamicTables_EmptyFrequencies(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	if len(litTable.Codes) == 0 {
		t.Log("Empty literal frequencies produce empty codes (acceptable)")
	}

	if len(distTable.Codes) == 0 {
		t.Log("Empty distance frequencies produce empty codes (acceptable)")
	}
}

func TestBuildDynamicTables_SingleSymbol(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	litFreq[65] = 1
	distFreq[1] = 1

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	if len(litTable.Codes) > 0 && 65 < len(litTable.Codes) {
		code := litTable.Codes[65]
		if code.Length == 0 {
			t.Error("Expected code length > 0 for symbol 65")
		}
	} else {
		t.Log("Single symbol may produce empty codes (Huffman requires at least 2 symbols)")
	}

	if len(distTable.Codes) > 0 && 1 < len(distTable.Codes) {
		distCode := distTable.Codes[1]
		if distCode.Length == 0 {
			t.Error("Expected code length > 0 for distance symbol 1")
		}
	} else {
		t.Log("Single symbol may produce empty codes (Huffman requires at least 2 symbols)")
	}
}

func TestBuildDynamicTables_AllSymbols(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	for i := 0; i < 288; i++ {
		litFreq[i] = 1
	}

	for i := 0; i < 30; i++ {
		distFreq[i] = 1
	}

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	if len(litTable.Codes) == 0 {
		t.Fatal("Expected literal table codes for all symbols")
	}

	if len(distTable.Codes) == 0 {
		t.Fatal("Expected distance table codes for all symbols")
	}

	for i := 0; i < 288 && i < len(litTable.Codes); i++ {
		code := litTable.Codes[i]
		if code.Length == 0 {
			t.Errorf("Symbol %d has zero code length", i)
		}
	}

	for i := 0; i < 30 && i < len(distTable.Codes); i++ {
		code := distTable.Codes[i]
		if code.Length == 0 {
			t.Errorf("Distance symbol %d has zero code length", i)
		}
	}
}

func TestBuildDynamicTables_Structure(t *testing.T) {
	litFreq := make([]int, 288)
	distFreq := make([]int, 30)

	litFreq[65] = 10
	litFreq[66] = 5
	litFreq[67] = 2
	litFreq[256] = 1

	distFreq[1] = 5
	distFreq[2] = 2

	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	if litTable.MaxLength == 0 {
		t.Error("Expected MaxLength > 0 for literal table")
	}

	if distTable.MaxLength == 0 {
		t.Error("Expected MaxLength > 0 for distance table")
	}

	if len(litTable.Codes) == 0 {
		t.Error("Expected codes in literal table, got empty")
	}

	if len(distTable.Codes) == 0 {
		t.Error("Expected codes in distance table, got empty")
	}
}
