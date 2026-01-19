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
