package png

func SelectFilter(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
	var bestFilter FilterType
	var bestFiltered []byte
	bestScore := -1

	type filterFunc func() []byte
	filters := []struct {
		typ FilterType
		fn  filterFunc
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
