package compress

// EstimateCompressedSize estimates the DEFLATE compressed size using a simplified LZ77 algorithm.
// This is much faster than full DEFLATE encoding but accurate enough for filter comparison.
// Returns: estimated size in bytes, number of matches found
func EstimateCompressedSize(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}

	window := NewSlidingWindow(32768) // 32KB DEFLATE window
	pos := 0
	estimatedSize := 0
	matches := 0

	for pos < len(data) {
		// Find longest match
		match, found := FindMatch(window, data, pos)

		if found && match.Length >= 3 {
			// Match: encoded as 2-3 bytes in DEFLATE (length + distance)
			estimatedSize += 3 // Conservative estimate
			matches++

			// Advance position by match length
			for i := 0; i < int(match.Length) && pos < len(data); i++ {
				window.Write(data[pos])
				pos++
			}
		} else {
			// Literal: 1 byte
			estimatedSize += 1
			window.Write(data[pos])
			pos++
		}
	}

	// Add DEFLATE block overhead (final block flag + block type)
	estimatedSize += 5

	return estimatedSize, matches
}
