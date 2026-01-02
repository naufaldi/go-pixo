package compress

const (
	hashBits = 15
	hashSize = 1 << hashBits
	hashMask = hashSize - 1
)

// LZ77Encoder encodes data using LZ77 compression with DEFLATE constraints.
type LZ77Encoder struct {
	head []int32
	prev []int32
}

// NewLZ77Encoder creates a new LZ77 encoder.
func NewLZ77Encoder() *LZ77Encoder {
	return &LZ77Encoder{
		head: make([]int32, hashSize),
	}
}

// Encode processes the input data and returns a sequence of tokens.
// Tokens are either literals or matches (back-references).
func (enc *LZ77Encoder) Encode(data []byte) []Token {
	if len(data) == 0 {
		return nil
	}

	// Initialize/reset hash table
	for i := range enc.head {
		enc.head[i] = -1
	}
	if len(enc.prev) < len(data) {
		enc.prev = make([]int32, len(data))
	}

	var tokens []Token
	pos := 0

	for pos < len(data) {
		remaining := len(data) - pos
		if remaining < minMatchLength {
			for pos < len(data) {
				tokens = append(tokens, TokenLiteral(data[pos]))
				pos++
			}
			break
		}

		// Find match using hash table
		match, found := enc.findMatch(data, pos)

		if found {
			tokens = append(tokens, TokenMatch(match.Distance, match.Length))
			// Update hash table for all bytes in the match
			for i := 0; i < int(match.Length); i++ {
				if pos+i+minMatchLength <= len(data) {
					h := enc.getHash(data[pos+i : pos+i+minMatchLength])
					enc.prev[pos+i] = enc.head[h]
					enc.head[h] = int32(pos + i)
				}
			}
			pos += int(match.Length)
		} else {
			// Update hash table for the literal byte
			h := enc.getHash(data[pos : pos+minMatchLength])
			enc.prev[pos] = enc.head[h]
			enc.head[h] = int32(pos)

			tokens = append(tokens, TokenLiteral(data[pos]))
			pos++
		}
	}

	return tokens
}

func (enc *LZ77Encoder) getHash(b []byte) uint32 {
	return (uint32(b[0])<<10 ^ uint32(b[1])<<5 ^ uint32(b[2])) & hashMask
}

func (enc *LZ77Encoder) findMatch(data []byte, pos int) (Match, bool) {
	h := enc.getHash(data[pos : pos+minMatchLength])
	matchPos := enc.head[h]

	bestLen := 0
	var bestMatch Match

	// Limit search depth to avoid O(N^2) in worst case
	chainLen := 0
	maxChainLen := 128

	for matchPos != -1 && chainLen < maxChainLen {
		dist := pos - int(matchPos)
		if dist > maxDistance {
			break
		}

		// Check match length
		matchLen := 0
		maxMatch := maxMatchLength
		if pos+maxMatch > len(data) {
			maxMatch = len(data) - pos
		}

		for matchLen < maxMatch && data[pos+matchLen] == data[int(matchPos)+matchLen] {
			matchLen++
		}

		if matchLen >= minMatchLength && matchLen > bestLen {
			bestLen = matchLen
			bestMatch = Match{
				Distance: uint16(dist),
				Length:   uint16(matchLen),
			}
			if bestLen >= maxMatchLength {
				break
			}
		}

		matchPos = enc.prev[matchPos]
		chainLen++
	}

	if bestLen >= minMatchLength {
		return bestMatch, true
	}
	return Match{}, false
}

