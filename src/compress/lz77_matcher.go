package compress

const (
	minMatchLength = 3
	maxMatchLength = 258
	maxDistance    = 32768
)

// FindMatch searches for the longest match starting at the current position
// in the lookahead buffer, looking back into the sliding window.
// Returns the best match found and true if a match of at least minMatchLength was found.
func FindMatch(window *SlidingWindow, lookahead []byte, lookaheadPos int) (Match, bool) {
	if len(lookahead) == 0 || lookaheadPos >= len(lookahead) {
		return Match{}, false
	}

	windowBytes := window.Bytes()
	if len(windowBytes) == 0 {
		return Match{}, false
	}

	maxLen := maxMatchLength
	if lookaheadPos+maxLen > len(lookahead) {
		maxLen = len(lookahead) - lookaheadPos
	}
	if maxLen < minMatchLength {
		return Match{}, false
	}

	bestMatch := Match{}
	bestLength := 0

	searchStart := lookahead[lookaheadPos:]
	maxSearchDistance := len(windowBytes)
	if maxSearchDistance > maxDistance {
		maxSearchDistance = maxDistance
	}

	for dist := 1; dist <= maxSearchDistance && dist <= len(windowBytes); dist++ {
		windowStart := len(windowBytes) - dist
		if windowStart < 0 {
			continue
		}

		matchLen := 0
		for matchLen < maxLen && matchLen < len(searchStart) {
			windowIdx := windowStart + matchLen
			if windowIdx >= len(windowBytes) {
				break
			}
			if windowBytes[windowIdx] != searchStart[matchLen] {
				break
			}
			matchLen++
		}

		if matchLen >= minMatchLength && matchLen > bestLength {
			bestLength = matchLen
			bestMatch = Match{
				Distance: uint16(dist),
				Length:   uint16(matchLen),
			}
		}
	}

	if bestLength >= minMatchLength {
		return bestMatch, true
	}
	return Match{}, false
}
