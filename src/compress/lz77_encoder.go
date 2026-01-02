package compress

const (
	hashBits = 15
	hashSize = 1 << hashBits
	hashMask = hashSize - 1
)

// LZ77Encoder encodes data using LZ77 compression with DEFLATE constraints.
type LZ77Encoder struct {
	head              []int32
	prev              []int32
	compressionLevel  int
	maxChainLen       int
	minMatchLen       int
}

// NewLZ77Encoder creates a new LZ77 encoder.
func NewLZ77Encoder() *LZ77Encoder {
	return &LZ77Encoder{
		head:              make([]int32, hashSize),
		compressionLevel:  6,
		maxChainLen:       128,
		minMatchLen:       minMatchLength,
	}
}

// SetCompressionLevel sets the compression level (1-9).
// Higher levels produce better compression but are slower.
func (enc *LZ77Encoder) SetCompressionLevel(level int) {
	if level < 1 {
		level = 1
	} else if level > 9 {
		level = 9
	}
	enc.compressionLevel = level

	switch level {
	case 1:
		enc.maxChainLen = 4
		enc.minMatchLen = 3
	case 2:
		enc.maxChainLen = 8
		enc.minMatchLen = 3
	case 3:
		enc.maxChainLen = 16
		enc.minMatchLen = 3
	case 4:
		enc.maxChainLen = 32
		enc.minMatchLen = 3
	case 5:
		enc.maxChainLen = 64
		enc.minMatchLen = 3
	case 6:
		enc.maxChainLen = 128
		enc.minMatchLen = 3
	case 7:
		enc.maxChainLen = 256
		enc.minMatchLen = 3
	case 8:
		enc.maxChainLen = 512
		enc.minMatchLen = 3
	case 9:
		enc.maxChainLen = 1024
		enc.minMatchLen = 3
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
		if remaining < enc.minMatchLen {
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
				if pos+i+enc.minMatchLen <= len(data) {
					h := enc.getHash(data[pos+i : pos+i+enc.minMatchLen])
					enc.prev[pos+i] = enc.head[h]
					enc.head[h] = int32(pos + i)
				}
			}
			pos += int(match.Length)
		} else {
			// Update hash table for the literal byte
			h := enc.getHash(data[pos : pos+enc.minMatchLen])
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
	h := enc.getHash(data[pos : pos+enc.minMatchLen])
	matchPos := enc.head[h]

	bestLen := 0
	var bestMatch Match

	// Limit search depth to avoid O(N^2) in worst case
	chainLen := 0

	for matchPos != -1 && chainLen < enc.maxChainLen {
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

		if matchLen >= enc.minMatchLen && matchLen > bestLen {
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

	if bestLen >= enc.minMatchLen {
		return bestMatch, true
	}
	return Match{}, false
}

