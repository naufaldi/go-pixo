package png

const (
	DefaultMinSumThreshold   = 0.0
	DefaultEntropyThreshold  = 0.0
	DefaultMinFiltersToTry   = 2
)

type FilterSelectionConfig struct {
	EarlyTerminationEnabled   bool
	FilterSelectionThreshold  float64
	EarlyTerminationThreshold float64
	MinFiltersToTry           int
}

func SelectAllWithScratch(pixels []byte, scratch *AdaptiveScratch) []FilterType {
	return SelectAllWithStrategyAndScratch(
		pixels,
		scratch.RowLen()/scratch.BPP(),
		len(pixels)/scratch.RowLen(),
		scratch.BPP(),
		FilterStrategyAdaptive,
		scratch,
	)
}

func SelectAllWithStrategyAndScratch(pixels []byte, width, height, bpp int, strategy FilterStrategy, scratch *AdaptiveScratch) []FilterType {
	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, strategy, scratch)
		filters[y] = filterType
		prevRow = row
	}

	return filters
}

func SelectFilterWithStrategyAndScratch(row []byte, prevRow []byte, bpp int, strategy FilterStrategy, scratch *AdaptiveScratch) (FilterType, []byte) {
	switch strategy {
	case FilterStrategyNone:
		filtered := ApplyFilterNoneWithScratch(row, scratch)
		return FilterNone, filtered[1:]
	case FilterStrategySub:
		filtered := ApplyFilterSubWithScratch(row, bpp, scratch)
		return FilterSub, filtered[1:]
	case FilterStrategyUp:
		filtered := ApplyFilterUpWithScratch(row, prevRow, scratch)
		return FilterUp, filtered[1:]
	case FilterStrategyAverage:
		filtered := ApplyFilterAverageWithScratch(row, prevRow, bpp, scratch)
		return FilterAverage, filtered[1:]
	case FilterStrategyPaeth:
		filtered := ApplyFilterPaethWithScratch(row, prevRow, bpp, scratch)
		return FilterPaeth, filtered[1:]
	case FilterStrategyMinSum:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyAdaptive:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyAdaptiveFast:
		return selectAdaptiveFast(row, prevRow, bpp)
	case FilterStrategyEntropy:
		return selectEntropyWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyBruteForce:
		return selectBruteForce(row, prevRow, bpp)
	case FilterStrategyBigrams:
		return selectBigrams(row, prevRow, bpp)
	case FilterStrategyParallel:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	default:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	}
}

func selectMinSumWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNoneWithScratch(row, scratch) }},
		{FilterSub, func() []byte { return ApplyFilterSubWithScratch(row, bpp, scratch) }},
		{FilterUp, func() []byte { return ApplyFilterUpWithScratch(row, prevRow, scratch) }},
		{FilterAverage, func() []byte { return ApplyFilterAverageWithScratch(row, prevRow, bpp, scratch) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaethWithScratch(row, prevRow, bpp, scratch) }},
	}

	minFilters := scratch.Config.MinFiltersToTry
	if minFilters <= 0 {
		minFilters = DefaultMinFiltersToTry
	}

	for i, f := range filters {
		filtered := f.fn()
		score := SumAbsoluteValues(filtered[1:])
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = filtered
		}

		if scratch.Config.EarlyTerminationEnabled &&
			i+1 >= minFilters &&
			float64(bestScore) <= scratch.Config.FilterSelectionThreshold {
			break
		}
	}

	return bestFilter, bestFiltered[1:]
}

func selectEntropyWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1.0

	filters := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNoneWithScratch(row, scratch) }},
		{FilterSub, func() []byte { return ApplyFilterSubWithScratch(row, bpp, scratch) }},
		{FilterUp, func() []byte { return ApplyFilterUpWithScratch(row, prevRow, scratch) }},
		{FilterAverage, func() []byte { return ApplyFilterAverageWithScratch(row, prevRow, bpp, scratch) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaethWithScratch(row, prevRow, bpp, scratch) }},
	}

	minFilters := scratch.Config.MinFiltersToTry
	if minFilters <= 0 {
		minFilters = DefaultMinFiltersToTry
	}

	for i, f := range filters {
		filtered := f.fn()
		score := EntropyScore(filtered[1:])
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = filtered
		}

		if scratch.Config.EarlyTerminationEnabled &&
			i+1 >= minFilters &&
			bestScore <= scratch.Config.EarlyTerminationThreshold {
			break
		}
	}

	return bestFilter, bestFiltered[1:]
}
