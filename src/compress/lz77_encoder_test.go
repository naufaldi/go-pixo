package compress

import (
	"testing"
)

func TestLZ77EncoderEmpty(t *testing.T) {
	enc := NewLZ77Encoder()
	tokens := enc.Encode([]byte{})
	if len(tokens) != 0 {
		t.Errorf("Encode([]byte{}) returned %d tokens, want 0", len(tokens))
	}
}

func TestLZ77EncoderNoMatches(t *testing.T) {
	enc := NewLZ77Encoder()
	data := []byte("ABCDEFGHIJKLMNOP")
	tokens := enc.Encode(data)
	if len(tokens) != len(data) {
		t.Errorf("Encode returned %d tokens, want %d", len(tokens), len(data))
	}
	for i, tok := range tokens {
		if !tok.IsLiteral {
			t.Errorf("Token[%d] is match, want literal", i)
		}
		if tok.Literal != data[i] {
			t.Errorf("Token[%d].Literal = %c, want %c", i, tok.Literal, data[i])
		}
	}
}

func TestLZ77EncoderSimpleRepeat(t *testing.T) {
	enc := NewLZ77Encoder()
	data := []byte("ABCABCABC")
	tokens := enc.Encode(data)

	literalCount := 0
	matchCount := 0
	for _, tok := range tokens {
		if tok.IsLiteral {
			literalCount++
		} else {
			matchCount++
			if tok.Match.Length < minMatchLength {
				t.Errorf("Match.Length = %d, want >= %d", tok.Match.Length, minMatchLength)
			}
		}
	}

	if literalCount == 0 {
		t.Error("Expected at least some literal tokens")
	}
	if matchCount == 0 {
		t.Error("Expected at least one match token for repeating pattern")
	}
}

func TestLZ77EncoderBoundaryConditions(t *testing.T) {
	enc := NewLZ77Encoder()

	t.Run("minLength", func(t *testing.T) {
		enc2 := NewLZ77Encoder()
		data := []byte("ABCABC")
		tokens := enc2.Encode(data)
		foundMatch := false
		for _, tok := range tokens {
			if !tok.IsLiteral {
				foundMatch = true
				if tok.Match.Length < minMatchLength {
					t.Errorf("Match.Length = %d, want >= %d", tok.Match.Length, minMatchLength)
				}
			}
		}
		if !foundMatch {
			t.Error("Expected match token for 'ABCABC'")
		}
	})

	t.Run("maxLength", func(t *testing.T) {
		enc3 := NewLZ77Encoder()
		pattern := make([]byte, 300)
		for i := range pattern {
			pattern[i] = byte('A')
		}
		tokens := enc3.Encode(pattern)
		for _, tok := range tokens {
			if !tok.IsLiteral {
				if tok.Match.Length > maxMatchLength {
					t.Errorf("Match.Length = %d, exceeds max %d", tok.Match.Length, maxMatchLength)
				}
			}
		}
	})

	_ = enc
}

func TestLZ77EncoderWindowUpdate(t *testing.T) {
	enc := NewLZ77Encoder()
	data := []byte("ABCABCABC")
	tokens1 := enc.Encode(data)

	enc2 := NewLZ77Encoder()
	tokens2 := enc2.Encode(data)

	if len(tokens1) != len(tokens2) {
		t.Errorf("Different token counts: %d vs %d", len(tokens1), len(tokens2))
	}

	for i := range tokens1 {
		if tokens1[i].IsLiteral != tokens2[i].IsLiteral {
			t.Errorf("Token[%d] type mismatch", i)
		}
		if tokens1[i].IsLiteral {
			if tokens1[i].Literal != tokens2[i].Literal {
				t.Errorf("Token[%d].Literal mismatch: %c vs %c", i, tokens1[i].Literal, tokens2[i].Literal)
			}
		} else {
			if tokens1[i].Match.Distance != tokens2[i].Match.Distance ||
				tokens1[i].Match.Length != tokens2[i].Match.Length {
				t.Errorf("Token[%d].Match mismatch", i)
			}
		}
	}
}
