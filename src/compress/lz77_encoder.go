package compress

// LZ77Encoder encodes data using LZ77 compression with DEFLATE constraints.
type LZ77Encoder struct {
	window *SlidingWindow
}

// NewLZ77Encoder creates a new LZ77 encoder with a 32KB sliding window.
func NewLZ77Encoder() *LZ77Encoder {
	return &LZ77Encoder{
		window: NewSlidingWindow(maxDistance),
	}
}

// Encode processes the input data and returns a sequence of tokens.
// Tokens are either literals or matches (back-references).
func (enc *LZ77Encoder) Encode(data []byte) []Token {
	if len(data) == 0 {
		return nil
	}

	// Each DEFLATE stream starts with an empty history window. The encoder is
	// reused across calls, so reset the sliding window to avoid producing matches
	// that reference bytes from previous encodings (which corrupts output).
	enc.window.Reset()

	var tokens []Token
	pos := 0

	for pos < len(data) {
		remaining := data[pos:]
		match, found := FindMatch(enc.window, data, pos)

		if found && match.Length <= uint16(len(remaining)) {
			tokens = append(tokens, TokenMatch(match.Distance, match.Length))
			for i := 0; i < int(match.Length); i++ {
				enc.window.Write(data[pos+i])
			}
			pos += int(match.Length)
		} else {
			tokens = append(tokens, TokenLiteral(data[pos]))
			enc.window.Write(data[pos])
			pos++
		}
	}

	return tokens
}
