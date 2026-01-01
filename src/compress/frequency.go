package compress

const (
	endOfBlockSymbol = 256
	maxLiteralSymbol = 255
	maxDistanceCode  = 29
)

// CountFrequencies counts the frequency of each literal symbol (0-255) and adds
// one for the end-of-block symbol (256).
func CountFrequencies(data []byte) []int {
	freq := make([]int, endOfBlockSymbol+1)
	for _, b := range data {
		freq[int(b)]++
	}
	freq[endOfBlockSymbol] = 1
	return freq
}

// CountDistanceFrequencies counts the frequency of each distance code from matches.
// Distance codes are 0-29 per DEFLATE spec.
func CountDistanceFrequencies(matches []Match) []int {
	freq := make([]int, maxDistanceCode+1)
	for _, m := range matches {
		code := distanceCode(m.Distance)
		if code <= maxDistanceCode {
			freq[code]++
		}
	}
	return freq
}

// distanceCode maps a distance value to its DEFLATE distance code (0-29).
// This is a simplified version; full implementation would use extra bits.
// Distance 0 is invalid in DEFLATE and returns -1.
func distanceCode(distance uint16) int {
	if distance == 0 {
		return -1
	}
	if distance <= 4 {
		return int(distance) - 1
	}
	if distance <= 8 {
		return 4
	}
	if distance <= 16 {
		return 5
	}
	if distance <= 32 {
		return 6
	}
	if distance <= 64 {
		return 7
	}
	if distance <= 128 {
		return 8
	}
	if distance <= 256 {
		return 9
	}
	if distance <= 512 {
		return 10
	}
	if distance <= 1024 {
		return 11
	}
	if distance <= 2048 {
		return 12
	}
	if distance <= 4096 {
		return 13
	}
	if distance <= 8192 {
		return 14
	}
	if distance <= 16384 {
		return 15
	}
	return 16
}
