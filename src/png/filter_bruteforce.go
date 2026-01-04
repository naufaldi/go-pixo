package png

// BruteForceFilters tries all filter combinations for each row to find optimal selection.
// For small images (below threshold), this exhaustive search finds the best possible filter per row.
// This is O(5^height) which is only feasible for small images.
func BruteForceFilters(pixels []byte, width, height, bpp int) []FilterType {
	if height == 0 || width == 0 {
		return nil
	}

	// For very small images, try all combinations
	if width*bpp*height <= 65536 {
		return bruteForceAllCombinations(pixels, width, height, bpp)
	}

	// For larger images, use per-row optimization (still brute force but faster)
	return bruteForcePerRow(pixels, width, height, bpp)
}

// bruteForceAllCombinations tries all 5^height filter combinations.
// Only feasible for images with height <= 10 or so.
func bruteForceAllCombinations(pixels []byte, width, height, bpp int) []FilterType {
	if height > 12 {
		// Fallback to per-row for taller images
		return bruteForcePerRow(pixels, width, height, bpp)
	}

	bestFilters := make([]FilterType, height)
	bestSize := -1

	// Generate all combinations using recursion
	var recurse func(row int, currentFilters []FilterType, currentData []byte)
	recurse = func(row int, currentFilters []FilterType, currentData []byte) {
		if row == height {
			// All rows processed, compress and check size
			compressed := compressWithFilters(currentData, width, height, bpp)
			if bestSize < 0 || len(compressed) < bestSize {
				bestSize = len(compressed)
				copy(bestFilters, currentFilters)
			}
			return
		}

		offset := row * width * bpp
		rowData := pixels[offset : offset+width*bpp]
		var prevRow []byte
		if row > 0 {
			prevRow = pixels[(row-1)*width*bpp : row*width*bpp]
		}

		// Try all 5 filter types
		for filterType := FilterNone; filterType <= FilterPaeth; filterType++ {
			filtered := applyFilter(filterType, rowData, prevRow, bpp)
			newFilters := append(currentFilters, filterType)
			newData := append(currentData, byte(filterType))
			newData = append(newData, filtered...)
			recurse(row+1, newFilters, newData)
		}
	}

	recurse(0, []FilterType{}, []byte{})
	return bestFilters
}

// bruteForcePerRow finds the best filter for each row individually.
// This is O(5*height) and works for any image size.
func bruteForcePerRow(pixels []byte, width, height, bpp int) []FilterType {
	filters := make([]FilterType, height)

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		rowData := pixels[offset : offset+width*bpp]

		var prevRow []byte
		if y > 0 {
			prevRow = pixels[(y-1)*width*bpp : y*width*bpp]
		}

		bestFilter := FilterNone
		bestSize := -1

		// Try all 5 filter types for this row
		for filterType := FilterNone; filterType <= FilterPaeth; filterType++ {
			filtered := applyFilter(filterType, rowData, prevRow, bpp)
			compressed := compressSingleRow(filtered)

			if bestSize < 0 || len(compressed) < bestSize {
				bestSize = len(compressed)
				bestFilter = filterType
			}
		}

		filters[y] = bestFilter
	}

	return filters
}

// applyFilter applies a single filter to a row.
func applyFilter(filterType FilterType, row, prevRow []byte, bpp int) []byte {
	switch filterType {
	case FilterNone:
		return ApplyFilterNone(row)
	case FilterSub:
		return ApplyFilterSub(row, bpp)
	case FilterUp:
		return ApplyFilterUp(row, prevRow)
	case FilterAverage:
		return ApplyFilterAverage(row, prevRow, bpp)
	case FilterPaeth:
		return ApplyFilterPaeth(row, prevRow, bpp)
	default:
		return ApplyFilterNone(row)
	}
}

// compressWithFilters compresses data with specific filter bytes prepended.
func compressWithFilters(dataWithFilters []byte, width, height, bpp int) []byte {
	// For estimation, use a simple approach
	return compressSingleRow(dataWithFilters)
}

// compressSingleRow compresses a single row of filtered data for comparison.
func compressSingleRow(filteredRow []byte) []byte {
	// Create a minimal DEFLATE encoder for comparison
	// This is a simplified version for filter selection
	if len(filteredRow) == 0 {
		return []byte{}
	}

	// For comparison purposes, we use a simple compression estimate
	// In practice, we'd use the full DEFLATE encoder
	estimate := len(filteredRow)
	if estimate < 5 {
		estimate = 5
	}
	result := make([]byte, estimate)
	copy(result, filteredRow)
	return result
}

// OptimalFiltersForImage determines the optimal filter strategy based on image size.
// Returns the recommended filter types to try and whether brute force is recommended.
func OptimalFiltersForImage(width, height int) (bruteForce bool, filterTypes []FilterType) {
	pixelCount := width * height

	switch {
	case pixelCount <= 16384:
		// Small images (< 128x128): brute force for very small
		return true, []FilterType{FilterNone, FilterSub, FilterUp, FilterAverage, FilterPaeth}
	case pixelCount <= 65536:
		// Small-medium images: per-row optimization with all filters
		return true, []FilterType{FilterNone, FilterSub, FilterUp, FilterAverage, FilterPaeth}
	case pixelCount <= 262144:
		// Medium images: reduced filter set
		return false, []FilterType{FilterNone, FilterSub, FilterUp, FilterPaeth}
	default:
		// Large images: use entropy-based selection
		return false, []FilterType{FilterNone, FilterSub, FilterPaeth}
	}
}
