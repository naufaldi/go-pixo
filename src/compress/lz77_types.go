package compress

// Match represents a back-reference to previously seen data.
// Distance is the number of bytes back to look, Length is how many bytes to copy.
type Match struct {
	Distance uint16
	Length   uint16
}

// Token represents either a literal byte or a match (back-reference).
// For literals, use TokenLiteral with the byte value.
// For matches, use TokenMatch with the Match struct.
type Token struct {
	IsLiteral bool
	Literal   byte
	Match     Match
}

// TokenLiteral creates a literal token.
func TokenLiteral(b byte) Token {
	return Token{IsLiteral: true, Literal: b}
}

// TokenMatch creates a match token.
func TokenMatch(distance, length uint16) Token {
	return Token{
		IsLiteral: false,
		Match: Match{
			Distance: distance,
			Length:   length,
		},
	}
}
