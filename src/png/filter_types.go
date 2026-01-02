package png

type FilterType uint8

const (
	FilterNone    FilterType = 0
	FilterSub     FilterType = 1
	FilterUp      FilterType = 2
	FilterAverage FilterType = 3
	FilterPaeth   FilterType = 4
)
