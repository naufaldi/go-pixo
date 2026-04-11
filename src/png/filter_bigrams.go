package png

func countDistinctBigrams(data []byte) int {
	if len(data) < 2 {
		return 0
	}

	bigrams := make(map[uint16]struct{}, len(data)-1)
	for i := 0; i < len(data)-1; i++ {
		bigram := uint16(data[i])<<8 | uint16(data[i+1])
		bigrams[bigram] = struct{}{}
	}
	return len(bigrams)
}

func selectBigrams(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestBigramCount := -1

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
		bigramCount := countDistinctBigrams(filtered)
		if bestBigramCount < 0 || bigramCount < bestBigramCount {
			bestBigramCount = bigramCount
			bestFilter = f.typ
			bestFiltered = filtered
		}
	}

	return bestFilter, bestFiltered
}

func SelectFilterBigrams(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	return selectBigrams(row, prevRow, bpp)
}

func SelectAllBigrams(pixels []byte, width, height, bpp int) []FilterType {
	filters := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := SelectFilterBigrams(row, prevRow, bpp)
		filters[y] = filterType
		prevRow = row
	}

	return filters
}

func ApplyBigrams(row []byte, prevRow []byte, bpp int) []byte {
	_, filtered := selectBigrams(row, prevRow, bpp)
	return filtered
}

// selectCombined picks the filter with the lowest combined Entropy + Bigrams rank.
// Each metric is normalized to [0,1] then summed; the filter with the smallest sum wins.
// When all filters score equally on one metric (range = 0), that metric contributes 0,
// so the algorithm naturally degrades to the other metric alone.
func selectCombined(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	type candidate struct {
		typ      FilterType
		filtered []byte
		entropy  float64
		bigrams  int
	}

	filterDefs := []struct {
		typ FilterType
		fn  func() []byte
	}{
		{FilterNone, func() []byte { return ApplyFilterNone(row) }},
		{FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
		{FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
		{FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
		{FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
	}

	candidates := make([]candidate, len(filterDefs))
	minE, maxE := -1.0, -1.0
	minB, maxB := -1, -1

	for i, f := range filterDefs {
		filtered := f.fn()
		e := CalculateEntropy(filtered)
		b := countDistinctBigrams(filtered)
		candidates[i] = candidate{f.typ, filtered, e, b}
		if minE < 0 || e < minE {
			minE = e
		}
		if maxE < 0 || e > maxE {
			maxE = e
		}
		if minB < 0 || b < minB {
			minB = b
		}
		if maxB < 0 || b > maxB {
			maxB = b
		}
	}

	eRange := maxE - minE
	bRange := float64(maxB - minB)
	bestIdx, bestScore := 0, 2.0

	for i, c := range candidates {
		var eN, bN float64
		if eRange > 0 {
			eN = (c.entropy - minE) / eRange
		}
		if bRange > 0 {
			bN = float64(c.bigrams-minB) / bRange
		}
		if s := eN + bN; s < bestScore {
			bestScore = s
			bestIdx = i
		}
	}

	return candidates[bestIdx].typ, candidates[bestIdx].filtered
}
