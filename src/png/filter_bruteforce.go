package png

import (
	"bytes"
	"compress/flate"
)

// BruteForceFilters compares five independent compressed scanlines per row.
// Work is linear in the image size; it never explores 5^height combinations.
// Row costs do not model the final compressor's cross-row history.
func BruteForceFilters(pixels []byte, width, height, bpp int) []FilterType {
	return BruteForceFiltersWithScratch(pixels, width, height, bpp)
}

// BruteForceFiltersWithScratch reuses one scanline scratch buffer across the image.
func BruteForceFiltersWithScratch(pixels []byte, width, height, bpp int) []FilterType {
	if height <= 0 || width <= 0 {
		return nil
	}
	scratch := NewAdaptiveScratch(width*bpp, bpp)
	return SelectAllWithStrategyAndScratch(pixels, width, height, bpp, FilterStrategyBruteForce, scratch)
}

// selectBruteForceWithScratch measures actual BestSpeed DEFLATE bytes, including
// the filter byte. The writer is reset between exactly five independent trials.
func selectBruteForceWithScratch(row, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var output bytes.Buffer
	writer, err := flate.NewWriter(&output, flate.BestSpeed)
	if err != nil {
		panic(err)
	} // The constant level is valid.
	bestFilter, bestSize := FilterNone, -1
	for candidate := FilterNone; candidate <= FilterPaeth; candidate++ {
		output.Reset()
		writer.Reset(&output)
		filtered := applyFilterWithScratch(candidate, row, prevRow, bpp, scratch)
		if _, err := writer.Write(filtered); err != nil {
			panic(err)
		} // bytes.Buffer cannot fail.
		if err := writer.Close(); err != nil {
			panic(err)
		}
		if bestSize < 0 || output.Len() < bestSize {
			bestFilter, bestSize = candidate, output.Len()
		}
	}
	return bestFilter, applyFilterWithScratch(bestFilter, row, prevRow, bpp, scratch)[1:]
}

// applyFilterWithScratch applies a single filter to a row using scratch buffers.
func applyFilterWithScratch(filterType FilterType, row, prevRow []byte, bpp int, scratch *AdaptiveScratch) []byte {
	switch filterType {
	case FilterNone:
		return ApplyFilterNoneWithScratch(row, scratch)
	case FilterSub:
		return ApplyFilterSubWithScratch(row, bpp, scratch)
	case FilterUp:
		return ApplyFilterUpWithScratch(row, prevRow, scratch)
	case FilterAverage:
		return ApplyFilterAverageWithScratch(row, prevRow, bpp, scratch)
	case FilterPaeth:
		return ApplyFilterPaethWithScratch(row, prevRow, bpp, scratch)
	default:
		return ApplyFilterNoneWithScratch(row, scratch)
	}
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

// compressSingleRow returns actual DEFLATE bytes for standalone row comparisons.
func compressSingleRow(filteredRow []byte) []byte {
	if len(filteredRow) == 0 {
		return nil
	}
	var output bytes.Buffer
	writer, err := flate.NewWriter(&output, flate.BestSpeed)
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write(filteredRow); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return output.Bytes()
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
