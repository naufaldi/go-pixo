package png

import (
	"bytes"
	"testing"
)

func TestFilterWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name string
		row  []byte
		prev []byte
		bpp  int
	}{
		{"grayscale", []byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1},
		{"rgb", []byte{100, 150, 200, 110, 160, 210}, []byte{50, 100, 150, 60, 110, 160}, 3},
		{"rgba", []byte{100, 150, 200, 255, 110, 160, 210, 255}, []byte{50, 100, 150, 255, 60, 110, 160, 255}, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scratch := NewAdaptiveScratch(len(tc.row), tc.bpp)
			defer scratch.Release()

			filteredOriginal := ApplyFilterNone(tc.row)
			filteredScratch := ApplyFilterNoneWithScratch(tc.row, scratch)

			if !bytes.Equal(filteredOriginal, filteredScratch[1:]) {
				t.Errorf("FilterNone output mismatch:\noriginal: %v\nscratch: %v", filteredOriginal, filteredScratch[1:])
			}
		})
	}
}

func TestFilterSubWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name string
		row  []byte
		bpp  int
	}{
		{"grayscale bpp=1", []byte{100, 150, 200, 250}, 1},
		{"rgb bpp=3", []byte{100, 150, 200, 110, 160, 210}, 3},
		{"rgba bpp=4", []byte{100, 150, 200, 255, 110, 160, 210, 255}, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scratch := NewAdaptiveScratch(len(tc.row), tc.bpp)
			defer scratch.Release()

			filteredOriginal := ApplyFilterSub(tc.row, tc.bpp)
			filteredScratch := ApplyFilterSubWithScratch(tc.row, tc.bpp, scratch)

			if !bytes.Equal(filteredOriginal, filteredScratch[1:]) {
				t.Errorf("FilterSub output mismatch:\noriginal: %v\nscratch: %v", filteredOriginal, filteredScratch[1:])
			}
		})
	}
}

func TestFilterUpWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name string
		row  []byte
		prev []byte
		bpp  int
	}{
		{"with prev row", []byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1},
		{"empty prev row", []byte{100, 150, 200, 250}, []byte{}, 1},
		{"rgb with prev", []byte{100, 150, 200, 110, 160, 210}, []byte{50, 100, 150, 60, 110, 160}, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scratch := NewAdaptiveScratch(len(tc.row), tc.bpp)
			defer scratch.Release()

			filteredOriginal := ApplyFilterUp(tc.row, tc.prev)
			filteredScratch := ApplyFilterUpWithScratch(tc.row, tc.prev, scratch)

			if !bytes.Equal(filteredOriginal, filteredScratch[1:]) {
				t.Errorf("FilterUp output mismatch:\noriginal: %v\nscratch: %v", filteredOriginal, filteredScratch[1:])
			}
		})
	}
}

func TestFilterAverageWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name string
		row  []byte
		prev []byte
		bpp  int
	}{
		{"with prev row", []byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1},
		{"empty prev row", []byte{100, 150, 200, 250}, []byte{}, 1},
		{"rgb with prev", []byte{100, 150, 200, 110, 160, 210}, []byte{50, 100, 150, 60, 110, 160}, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scratch := NewAdaptiveScratch(len(tc.row), tc.bpp)
			defer scratch.Release()

			filteredOriginal := ApplyFilterAverage(tc.row, tc.prev, tc.bpp)
			filteredScratch := ApplyFilterAverageWithScratch(tc.row, tc.prev, tc.bpp, scratch)

			if !bytes.Equal(filteredOriginal, filteredScratch[1:]) {
				t.Errorf("FilterAverage output mismatch:\noriginal: %v\nscratch: %v", filteredOriginal, filteredScratch[1:])
			}
		})
	}
}

func TestFilterPaethWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name string
		row  []byte
		prev []byte
		bpp  int
	}{
		{"with prev row", []byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1},
		{"empty prev row", []byte{100, 150, 200, 250}, []byte{}, 1},
		{"rgb with prev", []byte{100, 150, 200, 110, 160, 210}, []byte{50, 100, 150, 60, 110, 160}, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scratch := NewAdaptiveScratch(len(tc.row), tc.bpp)
			defer scratch.Release()

			filteredOriginal := ApplyFilterPaeth(tc.row, tc.prev, tc.bpp)
			filteredScratch := ApplyFilterPaethWithScratch(tc.row, tc.prev, tc.bpp, scratch)

			if !bytes.Equal(filteredOriginal, filteredScratch[1:]) {
				t.Errorf("FilterPaeth output mismatch:\noriginal: %v\nscratch: %v", filteredOriginal, filteredScratch[1:])
			}
		})
	}
}

func TestSelectAllWithScratch_Identity(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"small grayscale", 4, 4, 1},
		{"medium rgb", 8, 8, 3},
		{"large rgba", 16, 16, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pixels := make([]byte, tc.width*tc.height*tc.bpp)
			for i := range pixels {
				pixels[i] = byte(i % 256)
			}

			filtersOriginal := SelectAll(pixels, tc.width, tc.height, tc.bpp)

			scratch := NewAdaptiveScratch(tc.width*tc.bpp, tc.bpp)
			filtersScratch := SelectAllWithScratch(pixels, scratch)
			scratch.Release()

			if len(filtersOriginal) != len(filtersScratch) {
				t.Errorf("filter count mismatch: %d vs %d", len(filtersOriginal), len(filtersScratch))
			}

			for i := 0; i < len(filtersOriginal); i++ {
				if filtersOriginal[i] != filtersScratch[i] {
					t.Errorf("row %d: original=%d, scratch=%d", i, filtersOriginal[i], filtersScratch[i])
				}
			}
		})
	}
}

func TestSelectAllWithScratch_VariousStrategies(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	strategies := []FilterStrategy{
		FilterStrategyNone,
		FilterStrategySub,
		FilterStrategyUp,
		FilterStrategyAverage,
		FilterStrategyPaeth,
		FilterStrategyMinSum,
		FilterStrategyAdaptive,
		FilterStrategyAdaptiveFast,
	}

	for _, strategy := range strategies {
		t.Run(filterStrategyName(strategy), func(t *testing.T) {
			filtersOriginal := SelectAllWithStrategy(pixels, width, height, bpp, strategy)

			scratch := NewAdaptiveScratch(width*bpp, bpp)
			filtersScratch := SelectAllWithStrategy(pixels, width, height, bpp, strategy)
			scratch.Release()

			if len(filtersOriginal) != len(filtersScratch) {
				t.Errorf("filter count mismatch for strategy %s", strategyName(strategy))
			}

			for i := 0; i < len(filtersOriginal); i++ {
				if filtersOriginal[i] != filtersScratch[i] {
					t.Errorf("row %d: original=%d, scratch=%d", i, filtersOriginal[i], filtersScratch[i])
				}
			}
		})
	}
}

func filterStrategyName(s FilterStrategy) string {
	switch s {
	case FilterStrategyNone:
		return "None"
	case FilterStrategySub:
		return "Sub"
	case FilterStrategyUp:
		return "Up"
	case FilterStrategyAverage:
		return "Average"
	case FilterStrategyPaeth:
		return "Paeth"
	case FilterStrategyMinSum:
		return "MinSum"
	case FilterStrategyAdaptive:
		return "Adaptive"
	case FilterStrategyAdaptiveFast:
		return "AdaptiveFast"
	case FilterStrategyEntropy:
		return "Entropy"
	case FilterStrategyBruteForce:
		return "BruteForce"
	case FilterStrategyBigrams:
		return "Bigrams"
	case FilterStrategyParallel:
		return "Parallel"
	default:
		return "Unknown"
	}
}

func BenchmarkSelectAll_WithoutScratch(b *testing.B) {
	width, height, bpp := 256, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SelectAll(pixels, width, height, bpp)
	}
}

func BenchmarkSelectAll_WithScratch(b *testing.B) {
	width, height, bpp := 256, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		scratch := NewAdaptiveScratch(width*bpp, bpp)
		_ = SelectAllWithScratch(pixels, scratch)
		scratch.Release()
	}
}

func BenchmarkSelectAll_WithReusedScratch(b *testing.B) {
	width, height, bpp := 256, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	scratch := NewAdaptiveScratch(width*bpp, bpp)
	defer scratch.Release()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = SelectAllWithScratch(pixels, scratch)
	}
}

func BenchmarkFilterSelection_GCPressure(b *testing.B) {
	width, height, bpp := 512, 512, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.Run("without scratch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = SelectAll(pixels, width, height, bpp)
		}
	})

	b.Run("with scratch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			scratch := NewAdaptiveScratch(width*bpp, bpp)
			_ = SelectAllWithScratch(pixels, scratch)
			scratch.Release()
		}
	})

	b.Run("with reused scratch", func(b *testing.B) {
		scratch := NewAdaptiveScratch(width*bpp, bpp)
		defer scratch.Release()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = SelectAllWithScratch(pixels, scratch)
		}
	})
}

func BenchmarkBruteForceFilters_WithAndWithoutScratch(b *testing.B) {
	width, height, bpp := 32, 32, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	b.Run("without scratch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = BruteForceFilters(pixels, width, height, bpp)
		}
	})

	b.Run("with scratch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = BruteForceFiltersWithScratch(pixels, width, height, bpp)
		}
	})
}

func TestSelectAllWithStrategyAndScratch_Identity(t *testing.T) {
	width, height, bpp := 16, 16, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

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

	expectedFilters := SelectAll(pixels, width, height, bpp)

	if len(filters) != len(expectedFilters) {
		t.Errorf("filter count mismatch: %d vs %d", len(filters), len(expectedFilters))
	}

	for i := 0; i < len(filters); i++ {
		if filters[i] != expectedFilters[i] {
			t.Errorf("row %d: scratch=%d, expected=%d", i, filters[i], expectedFilters[i])
		}
	}
}

func TestSelectAllWithStrategyAndScratch_AllStrategies(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	scratch := NewAdaptiveScratch(width*bpp, bpp)
	defer scratch.Release()

	strategies := []FilterStrategy{
		FilterStrategyNone,
		FilterStrategySub,
		FilterStrategyUp,
		FilterStrategyAverage,
		FilterStrategyPaeth,
		FilterStrategyMinSum,
		FilterStrategyAdaptive,
		FilterStrategyAdaptiveFast,
	}

	for _, strategy := range strategies {
		t.Run(filterStrategyName(strategy), func(t *testing.T) {
			filters := make([]FilterType, height)
			var prevRow []byte

			for y := 0; y < height; y++ {
				offset := y * width * bpp
				row := pixels[offset : offset+width*bpp]
				filterType, _ := SelectFilterWithStrategyAndScratch(row, prevRow, bpp, strategy, scratch)
				filters[y] = filterType
				prevRow = row
			}

			expectedFilters := SelectAllWithStrategy(pixels, width, height, bpp, strategy)

			if len(filters) != len(expectedFilters) {
				t.Errorf("filter count mismatch for strategy %s", strategyName(strategy))
			}

			for i := 0; i < len(filters); i++ {
				if filters[i] != expectedFilters[i] {
					t.Errorf("strategy %s, row %d: scratch=%d, expected=%d", strategyName(strategy), i, filters[i], expectedFilters[i])
				}
			}
		})
	}
}
