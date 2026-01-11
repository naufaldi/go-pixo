package png

func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return SelectFilterWithStrategy(row, prevRow, bpp, FilterStrategyAdaptive)
}

func SelectFilterBigrams(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return selectBigrams(row, prevRow, bpp)
}

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
	case FilterStrategyBigrams:
		return selectBigrams(row, prevRow, bpp)
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
		compressed := compressSingleRow(filtered)
		size := len(compressed)
		if bestSize < 0 || size < bestSize {
			bestSize = size
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

func SelectAll(pixels []byte, width, height, bpp int) []FilterType {
	scratch := NewAdaptiveScratch(width*bpp, bpp)
	defer scratch.Release()

	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, FilterStrategyAdaptive, scratch)
		filters[y] = filterType

		prevRow = row
	}

	return filters
}

func SelectAllWithStrategy(pixels []byte, width, height, bpp int, strategy FilterStrategy) []FilterType {
	if strategy == FilterStrategyParallel {
		return SelectAllParallel(pixels, width, height, bpp)
	}

	scratch := NewAdaptiveScratch(width*bpp, bpp)
	defer scratch.Release()

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

func SelectAllWithBruteForce(pixels []byte, width, height, bpp int) []FilterType {
	return BruteForceFiltersWithScratch(pixels, width, height, bpp)
}

func SelectAllBigrams(pixels []byte, width, height, bpp int) []FilterType {
	scratch := NewAdaptiveScratch(width*bpp, bpp)
	defer scratch.Release()

	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, FilterStrategyBigrams, scratch)
		filters[y] = filterType

		prevRow = row
	}

	return filters
}

func SelectAllWithScratch(pixels []byte, scratch *AdaptiveScratch) []FilterType {
	rowLen := scratch.RowLen()
	bpp := scratch.BPP()
	width := rowLen / bpp
	height := len(pixels) / (width * bpp)

	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, FilterStrategyAdaptive, scratch)
		filters[y] = filterType

		prevRow = row
	}

	return filters
}

func SelectFilterWithStrategyAndScratch(row []byte, prevRow []byte, bpp int, strategy FilterStrategy, scratch *AdaptiveScratch) (FilterType, []byte) {
	switch strategy {
	case FilterStrategyNone:
		return selectNoneWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategySub:
		return selectSubWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyUp:
		return selectUpWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyAverage:
		return selectAverageWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyPaeth:
		return selectPaethWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyMinSum:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyAdaptive:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyAdaptiveFast:
		return selectAdaptiveFastWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyEntropy:
		return selectEntropyWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyBruteForce:
		return selectBruteForceWithScratch(row, prevRow, bpp, scratch)
	case FilterStrategyBigrams:
		return selectBigramsWithScratch(row, prevRow, bpp, scratch)
	default:
		return selectMinSumWithScratch(row, prevRow, bpp, scratch)
	}
}

func selectNoneWithScratch(row []byte, _ []byte, _ int, scratch *AdaptiveScratch) (FilterType, []byte) {
	return FilterNone, ApplyFilterNoneWithScratch(row, scratch)
}

func selectSubWithScratch(row []byte, _ []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	return FilterSub, ApplyFilterSubWithScratch(row, bpp, scratch)
}

func selectUpWithScratch(row []byte, prevRow []byte, _ int, scratch *AdaptiveScratch) (FilterType, []byte) {
	return FilterUp, ApplyFilterUpWithScratch(row, prevRow, scratch)
}

func selectAverageWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	return FilterAverage, ApplyFilterAverageWithScratch(row, prevRow, bpp, scratch)
}

func selectPaethWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	return FilterPaeth, ApplyFilterPaethWithScratch(row, prevRow, bpp, scratch)
}

func selectMinSumWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	filtered := scratch.GetFilteredRow()

	filters := []struct {
		typ FilterType
		fn  func(dst []byte) []byte
	}{
		{FilterNone, func(dst []byte) []byte { return ApplyFilterNoneWithScratchDst(row, dst) }},
		{FilterSub, func(dst []byte) []byte { return ApplyFilterSubWithScratchDst(row, bpp, dst) }},
		{FilterUp, func(dst []byte) []byte { return ApplyFilterUpWithScratchDst(row, prevRow, dst) }},
		{FilterAverage, func(dst []byte) []byte { return ApplyFilterAverageWithScratchDst(row, prevRow, bpp, dst) }},
		{FilterPaeth, func(dst []byte) []byte { return ApplyFilterPaethWithScratchDst(row, prevRow, bpp, dst) }},
	}

	for i, f := range filters {
		result := f.fn(filtered)
		score := SumAbsoluteValues(result[1:])
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = result
		}
		if scratch.Config.EarlyTerminationEnabled && i >= scratch.Config.MinFiltersToTry-1 {
			if float64(bestScore) <= scratch.Config.EarlyTerminationThreshold {
				return bestFilter, bestFiltered
			}
		}
	}

	return bestFilter, bestFiltered
}

func selectAdaptiveFastWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	filtered := scratch.GetFilteredRow()

	filters := []struct {
		typ FilterType
		fn  func(dst []byte) []byte
	}{
		{FilterNone, func(dst []byte) []byte { return ApplyFilterNoneWithScratchDst(row, dst) }},
		{FilterSub, func(dst []byte) []byte { return ApplyFilterSubWithScratchDst(row, bpp, dst) }},
		{FilterUp, func(dst []byte) []byte { return ApplyFilterUpWithScratchDst(row, prevRow, dst) }},
	}

	for _, f := range filters {
		result := f.fn(filtered)
		score := SumAbsoluteValues(result[1:])
		if bestScore < 0 || score < bestScore {
			bestScore = score
			bestFilter = f.typ
			bestFiltered = result
		}
	}

	return bestFilter, bestFiltered
}

func selectEntropyWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestEntropy := -1.0

	filtered := scratch.GetFilteredRow()

	filters := []struct {
		typ FilterType
		fn  func(dst []byte) []byte
	}{
		{FilterNone, func(dst []byte) []byte { return ApplyFilterNoneWithScratchDst(row, dst) }},
		{FilterSub, func(dst []byte) []byte { return ApplyFilterSubWithScratchDst(row, bpp, dst) }},
		{FilterUp, func(dst []byte) []byte { return ApplyFilterUpWithScratchDst(row, prevRow, dst) }},
		{FilterAverage, func(dst []byte) []byte { return ApplyFilterAverageWithScratchDst(row, prevRow, bpp, dst) }},
		{FilterPaeth, func(dst []byte) []byte { return ApplyFilterPaethWithScratchDst(row, prevRow, bpp, dst) }},
	}

	for i, f := range filters {
		result := f.fn(filtered)
		entropy := CalculateEntropy(result[1:])
		if bestEntropy < 0 || entropy < bestEntropy {
			bestEntropy = entropy
			bestFilter = f.typ
			bestFiltered = result
		}
		if scratch.Config.EarlyTerminationEnabled && i >= scratch.Config.MinFiltersToTry-1 {
			if bestEntropy <= scratch.Config.EarlyTerminationThreshold {
				return bestFilter, bestFiltered
			}
		}
	}

	return bestFilter, bestFiltered
}

func selectBigramsWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	filterType, _ := SelectFilterBigrams(row, prevRow, bpp)
	return filterType, ApplyFilterNoneWithScratch(row, scratch)
}

func selectBruteForceWithScratch(row []byte, prevRow []byte, bpp int, scratch *AdaptiveScratch) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestSize := -1

	filtered := scratch.GetFilteredRow()

	filters := []struct {
		typ FilterType
		fn  func(dst []byte) []byte
	}{
		{FilterNone, func(dst []byte) []byte { return ApplyFilterNoneWithScratchDst(row, dst) }},
		{FilterSub, func(dst []byte) []byte { return ApplyFilterSubWithScratchDst(row, bpp, dst) }},
		{FilterUp, func(dst []byte) []byte { return ApplyFilterUpWithScratchDst(row, prevRow, dst) }},
		{FilterAverage, func(dst []byte) []byte { return ApplyFilterAverageWithScratchDst(row, prevRow, bpp, dst) }},
		{FilterPaeth, func(dst []byte) []byte { return ApplyFilterPaethWithScratchDst(row, prevRow, bpp, dst) }},
	}

	for _, f := range filters {
		result := f.fn(filtered)
		compressed := compressSingleRow(result)
		size := len(compressed)
		if bestSize < 0 || size < bestSize {
			bestSize = size
			bestFilter = f.typ
			bestFiltered = result
		}
	}

	return bestFilter, bestFiltered
}
