package png

// SelectFilter picks a filter for a scanline using the default adaptive strategy.
func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return SelectFilterWithStrategy(row, prevRow, bpp, FilterStrategyAdaptive)
}

// SelectFilterWithStrategy picks a filter for a scanline based on the given strategy.
func SelectFilterWithStrategy(row []byte, prevRow []byte, bpp int, strategy FilterStrategy) (FilterType, []byte) {
	switch strategy {
	case FilterStrategyNone:
		return selectNone(row, prevRow, bpp)
	case FilterStrategySub:
		return selectSub(row, prevRow, bpp)
	case FilterStrategyUp:
		return selectUp(row, prevRow, bpp)
	case FilterStrategyAverage:
		return selectAverage(row, prevRow, bpp)
	case FilterStrategyPaeth:
		return selectPaeth(row, prevRow, bpp)
	case FilterStrategyMinSum:
		return selectMinSum(row, prevRow, bpp)
	case FilterStrategyAdaptive:
		return selectAdaptive(row, prevRow, bpp)
	case FilterStrategyAdaptiveFast:
		return selectAdaptiveFast(row, prevRow, bpp)
	case FilterStrategyEntropy:
		return selectEntropy(row, prevRow, bpp)
	case FilterStrategyBruteForce:
		return selectBruteForce(row, prevRow, bpp)
	default:
		return selectAdaptive(row, prevRow, bpp)
	}
}

func selectNone(row []byte, _ []byte, _ int) (FilterType, []byte) {
	return FilterNone, ApplyFilterNone(row)
}

func selectSub(row []byte, _ []byte, bpp int) (FilterType, []byte) {
	return FilterSub, ApplyFilterSub(row, bpp)
}

func selectUp(row []byte, prevRow []byte, _ int) (FilterType, []byte) {
	return FilterUp, ApplyFilterUp(row, prevRow)
}

func selectAverage(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return FilterAverage, ApplyFilterAverage(row, prevRow, bpp)
}

func selectPaeth(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return FilterPaeth, ApplyFilterPaeth(row, prevRow, bpp)
}

func selectMinSum(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNone(row) }},
		{FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
		{FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
		{FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
	}

	for _, f := range filters {
		filtered := f.fn()
		score := SumAbsoluteValues(filtered)
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

func selectAdaptive(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return selectMinSum(row, prevRow, bpp)
}

func selectAdaptiveFast(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	// Try a subset of filters for speed: None, Sub, Up
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNone(row) }},
		{FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
		{FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
	}

	for _, f := range filters {
		filtered := f.fn()
		score := SumAbsoluteValues(filtered)
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

func selectEntropy(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestEntropy := -1.0

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNone(row) }},
		{FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
		{FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
		{FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
	}

	for _, f := range filters {
		filtered := f.fn()
		entropy := CalculateEntropy(filtered)
		if bestEntropy < 0 || entropy < bestEntropy {
			bestEntropy = entropy
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

// SelectFilterWithEntropy picks a filter for a scanline by minimizing entropy.
func SelectFilterWithEntropy(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return selectEntropy(row, prevRow, bpp)
}

func selectBruteForce(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestSize := -1

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNone(row) }},
		{FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
		{FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
		{FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
	}

	for _, f := range filters {
		filtered := f.fn()
		size := len(filtered)
		if bestSize < 0 || size < bestSize {
			bestSize = size
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

// SelectAll returns the chosen filter type for each scanline using the default strategy.
func SelectAll(pixels []byte, width, height, bpp int) []FilterType {
	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilter(row, prevRow, bpp)
		filters[y] = filterType

		prevRow = row
	}

	return filters
}

// SelectAllWithStrategy returns the chosen filter type for each scanline using the provided strategy.
func SelectAllWithStrategy(pixels []byte, width, height, bpp int, strategy FilterStrategy) []FilterType {
	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterWithStrategy(row, prevRow, bpp, strategy)
		filters[y] = filterType

		prevRow = row
	}

	return filters
}

// SelectAllWithBruteForce applies brute force filter selection to all rows.
func SelectAllWithBruteForce(pixels []byte, width, height, bpp int) []FilterType {
	return BruteForceFilters(pixels, width, height, bpp)
}
