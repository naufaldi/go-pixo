package compress

import (
	"bytes"
	"compress/flate"
	"io"
	"testing"
)

func TestWriteStoredBlockDeflate(t *testing.T) {
	data := []byte("Hello, World!")
	var buf bytes.Buffer

	if err := WriteStoredBlockDeflate(&buf, true, data); err != nil {
		t.Fatalf("WriteStoredBlockDeflate failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("expected bytes written")
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, len(data))
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("got %d bytes, want %d", n, len(data))
	}

	if !bytes.Equal(decompressed[:n], data) {
		t.Errorf("got %q, want %q", decompressed[:n], data)
	}
}

func TestWriteFixedBlock_Simple(t *testing.T) {
	tokens := []Token{
		TokenLiteral('H'),
		TokenLiteral('e'),
		TokenLiteral('l'),
		TokenLiteral('l'),
		TokenLiteral('o'),
	}

	var buf bytes.Buffer
	if err := WriteFixedBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteFixedBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("Hello")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}

func TestWriteFixedBlock_WithMatch(t *testing.T) {
	tokens := []Token{
		TokenLiteral('A'),
		TokenLiteral('B'),
		TokenLiteral('C'),
		TokenMatch(3, 3),
	}

	var buf bytes.Buffer
	if err := WriteFixedBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteFixedBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("ABCABC")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}

func TestWriteDynamicBlock_Simple(t *testing.T) {
	tokens := []Token{
		TokenLiteral('H'),
		TokenLiteral('e'),
		TokenLiteral('l'),
		TokenLiteral('l'),
		TokenLiteral('o'),
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("Hello")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}

func TestWriteDynamicBlock_WithMatch(t *testing.T) {
	tokens := []Token{
		TokenLiteral('A'),
		TokenLiteral('B'),
		TokenLiteral('C'),
		TokenMatch(3, 3),
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("ABCABC")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}

func TestWriteDynamicBlock_SparseAlphabet(t *testing.T) {
	tokens := []Token{
		TokenLiteral('X'),
		TokenLiteral('Y'),
		TokenLiteral('Z'),
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("XYZ")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}

func TestWriteDynamicBlock_LongZeroRun(t *testing.T) {
	tokens := make([]Token, 20)
	for i := 0; i < 20; i++ {
		tokens[i] = TokenLiteral(byte('A' + (i % 26)))
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if n != 20 {
		t.Errorf("got %d bytes, want 20", n)
	}
}

func TestWriteFixedBlock_EndOfBlock(t *testing.T) {
	tokens := []Token{
		TokenLiteral('A'),
	}

	var buf bytes.Buffer
	if err := WriteFixedBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteFixedBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if n != 1 || decompressed[0] != 'A' {
		t.Errorf("got %q, want 'A'", decompressed[:n])
	}
}

func TestWriteDynamicBlock_BoundaryLengths(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"min length", 3},
		{"length 4", 4},
		{"length 258", 258},
		{"max length", 258},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := []Token{
				TokenLiteral('A'),
				TokenMatch(1, uint16(tt.length)),
			}

			var buf bytes.Buffer
			if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
				t.Fatalf("WriteDynamicBlock failed: %v", err)
			}

			reader := flate.NewReader(&buf)
			decompressed := make([]byte, 1000)
			n, err := reader.Read(decompressed)
			if err != nil && err != io.EOF {
				t.Fatalf("decompression failed: %v", err)
			}

			if n < 1+tt.length {
				t.Errorf("got %d bytes, want at least %d", n, 1+tt.length)
			}
		})
	}
}

func TestWriteDynamicBlock_BoundaryDistances(t *testing.T) {
	tests := []struct {
		name     string
		distance int
	}{
		{"min distance", 1},
		{"distance 2", 2},
		{"distance 32768", 32768},
		{"max distance", 32768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A match distance must not exceed the number of bytes already emitted in
			// the current block. Emit a prefix of length == distance, then a match.
			prefix := make([]Token, tt.distance)
			for i := range prefix {
				prefix[i] = TokenLiteral('A')
			}
			tokens := append(prefix, TokenMatch(uint16(tt.distance), 3))

			var buf bytes.Buffer
			if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
				t.Fatalf("WriteDynamicBlock failed: %v", err)
			}

			reader := flate.NewReader(&buf)
			t.Cleanup(func() { _ = reader.Close() })

			decompressed, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("decompression failed: %v", err)
			}

			if len(decompressed) < tt.distance+3 {
				t.Errorf("got %d bytes, want at least %d", len(decompressed), tt.distance+3)
			}
		})
	}
}

func TestWriteDynamicBlock_EOBMustHaveCode(t *testing.T) {
	tokens := []Token{
		TokenLiteral('A'),
	}

	litFreq, distFreq := countTokenFrequencies(tokens)
	litTable, _ := BuildDynamicTables(litFreq, distFreq)

	if litTable.Codes[EndOfBlockSymbol].Length == 0 {
		t.Fatalf("End-of-Block symbol (256) must have a non-zero code length, got 0")
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	if n != 1 || decompressed[0] != 'A' {
		t.Errorf("got %q, want 'A'", decompressed[:n])
	}
}

func TestWriteDynamicBlock_NoZeroLengthCodesForEmittedSymbols(t *testing.T) {
	tokens := []Token{
		TokenLiteral('A'),
		TokenLiteral('B'),
		TokenLiteral('C'),
		TokenMatch(3, 5),
	}

	litFreq, distFreq := countTokenFrequencies(tokens)
	litTable, distTable := BuildDynamicTables(litFreq, distFreq)

	matchLen := int(tokens[3].Match.Length)
	matchDist := int(tokens[3].Match.Distance)
	lengthSymbol := findLengthCode(matchLen)
	distSymbol := findDistanceCode(matchDist)

	symbolsToCheck := []struct {
		name  string
		table Table
		codes []int
	}{
		{"literal/length", litTable, []int{65, 66, 67, EndOfBlockSymbol, lengthSymbol}},
		{"distance", distTable, []int{distSymbol}},
	}

	for _, st := range symbolsToCheck {
		for _, symbol := range st.codes {
			if symbol >= len(st.table.Codes) {
				continue
			}
			code := st.table.Codes[symbol]
			if code.Length == 0 {
				t.Errorf("%s table: symbol %d has zero-length code (will cause 'invalid symbol' error)", st.name, symbol)
			}
		}
	}

	var buf bytes.Buffer
	if err := WriteDynamicBlock(&buf, true, tokens); err != nil {
		t.Fatalf("WriteDynamicBlock failed: %v", err)
	}

	reader := flate.NewReader(&buf)
	decompressed := make([]byte, 100)
	n, err := reader.Read(decompressed)
	if err != nil && err != io.EOF {
		t.Fatalf("decompression failed: %v", err)
	}

	expected := []byte("ABCABCAB")
	if !bytes.Equal(decompressed[:n], expected) {
		t.Errorf("got %q, want %q", decompressed[:n], expected)
	}
}
