package png

// FilterType identifies a PNG scanline filter.
type FilterType uint8

const (
	// FilterNone is the unfiltered scanline.
	FilterNone FilterType = 0
	// FilterSub is the Sub filter.
	FilterSub FilterType = 1
	// FilterUp is the Up filter.
	FilterUp FilterType = 2
	// FilterAverage is the Average filter.
	FilterAverage FilterType = 3
	// FilterPaeth is the Paeth filter.
	FilterPaeth FilterType = 4
)
