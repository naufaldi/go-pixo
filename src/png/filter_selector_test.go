package png

import "testing"

func TestSelectFilter(t *testing.T) {
	tests := []struct {
		name   string
		row    []byte
		prev   []byte
		bpp    int
		verify func(*testing.T, FilterType, []byte)
	}{
		{
			name: "returns valid filter type",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{0, 0, 0, 0},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if filterType < FilterNone || filterType > FilterPaeth {
					t.Errorf("filter type %d out of valid range [0-4]", filterType)
				}
			},
		},
		{
			name: "filtered length matches row length",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{0, 0, 0, 0},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 4 {
					t.Errorf("filtered length %d != row length 4", len(filtered))
				}
			},
		},
		{
			name: "deterministic selection",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{50, 100, 150, 200},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				filterType2, filtered2 := SelectFilter([]byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1)
				if filterType != filterType2 {
					t.Errorf("non-deterministic: first call %d, second call %d", filterType, filterType2)
				}
				if len(filtered) != len(filtered2) {
					t.Errorf("non-deterministic filtered length")
				}
			},
		},
		{
			name: "RGB bpp=3",
			row:  []byte{100, 150, 200, 110, 160, 210},
			prev: []byte{50, 100, 150, 60, 110, 160},
			bpp:  3,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 6 {
					t.Errorf("filtered length %d != row length 6", len(filtered))
				}
			},
		},
		{
			name: "RGBA bpp=4",
			row:  []byte{100, 150, 200, 255, 110, 160, 210, 255},
			prev: []byte{50, 100, 150, 255, 60, 110, 160, 255},
			bpp:  4,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 8 {
					t.Errorf("filtered length %d != row length 8", len(filtered))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterType, filtered := SelectFilter(tt.row, tt.prev, tt.bpp)
			tt.verify(t, filterType, filtered)
		})
	}
}

func TestSelectAll(t *testing.T) {
	width, height, bpp := 4, 3, 1
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i * 10)
	}

	filters := SelectAll(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("SelectAll returned %d filters, want %d", len(filters), height)
	}

	for i, f := range filters {
		if f < FilterNone || f > FilterPaeth {
			t.Errorf("filter[%d] = %d out of valid range [0-4]", i, f)
		}
	}
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantErr    bool
		minEntropy float64
		maxEntropy float64
	}{
		{
			name:       "empty data",
			data:       []byte{},
			wantErr:    false,
			minEntropy: 0,
			maxEntropy: 0,
		},
		{
			name:       "single byte repeated",
			data:       []byte{100, 100, 100, 100},
			wantErr:    false,
			minEntropy: 0,
			maxEntropy: 0.1,
		},
		{
			name:       "all different bytes",
			data:       []byte{0, 1, 2, 3, 4, 5, 6, 7},
			wantErr:    false,
			minEntropy: 2.5,
			maxEntropy: 8.0,
		},
		{
			name:       "random data high entropy",
			data:       []byte{0xAB, 0xCD, 0xEF, 0x12, 0x34, 0x56, 0x78, 0x9A},
			wantErr:    false,
			minEntropy: 2.5,
			maxEntropy: 8.0,
		},
		{
			name:       "two values alternating",
			data:       []byte{0, 255, 0, 255, 0, 255, 0, 255},
			wantErr:    false,
			minEntropy: 0.9,
			maxEntropy: 1.1,
		},
		{
			name:       "pattern data medium entropy",
			data:       []byte{0, 0, 1, 1, 2, 2, 3, 3, 4, 4},
			wantErr:    false,
			minEntropy: 1.5,
			maxEntropy: 4.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEntropy(tt.data)
			if tt.wantErr && got == 0 && len(tt.data) > 0 {
				t.Errorf("CalculateEntropy() = %v, want error", got)
			}
			if got < tt.minEntropy || got > tt.maxEntropy {
				t.Errorf("CalculateEntropy() = %v, want between %v and %v", got, tt.minEntropy, tt.maxEntropy)
			}
		})
	}
}

func TestEntropyScore(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "uniform data low score",
			data: []byte{50, 50, 50, 50, 50},
		},
		{
			name: "random data high score",
			data: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := EntropyScore(tt.data)
			entropy := CalculateEntropy(tt.data)
			expected := entropy * float64(len(tt.data))
			if score != expected {
				t.Errorf("EntropyScore() = %v, expected %v", score, expected)
			}
		})
	}
}

func TestSelectFilterWithEntropy(t *testing.T) {
	tests := []struct {
		name string
		row  []byte
		prev []byte
		bpp  int
	}{
		{
			name: "simple grayscale row",
			row:  []byte{100, 100, 100, 100},
			prev: []byte{100, 100, 100, 100},
			bpp:  1,
		},
		{
			name: "RGB row",
			row:  []byte{100, 150, 200, 110, 160, 210},
			prev: []byte{50, 100, 150, 60, 110, 160},
			bpp:  3,
		},
		{
			name: "RGBA row",
			row:  []byte{100, 150, 200, 255, 110, 160, 210, 255},
			prev: []byte{50, 100, 150, 255, 60, 110, 160, 255},
			bpp:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterType, filtered := SelectFilterWithEntropy(tt.row, tt.prev, tt.bpp)
			if filterType < FilterNone || filterType > FilterPaeth {
				t.Errorf("filter type %d out of valid range [0-4]", filterType)
			}
			if len(filtered) != len(tt.row) {
				t.Errorf("filtered length %d != row length %d", len(filtered), len(tt.row))
			}
		})
	}
}

func TestSelectFilterWithStrategyEntropy(t *testing.T) {
	row := []byte{100, 150, 200, 250, 100, 150, 200, 250}
	prev := []byte{50, 100, 150, 200, 60, 110, 160, 210}
	bpp := 4

	filterType, filtered := SelectFilterWithStrategy(row, prev, bpp, FilterStrategyEntropy)

	if filterType < FilterNone || filterType > FilterPaeth {
		t.Errorf("filter type %d out of valid range [0-4]", filterType)
	}
	if len(filtered) != len(row) {
		t.Errorf("filtered length %d != row length %d", len(filtered), len(row))
	}
}

func TestEntropyFilterStrategyVsMinSum(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 16)
			pixels[offset+1] = byte(y * 16)
			pixels[offset+2] = byte((x + y) * 8)
			pixels[offset+3] = 255
		}
	}

	filtersMinSum := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyMinSum)
	filtersEntropy := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyEntropy)

	if len(filtersMinSum) != height || len(filtersEntropy) != height {
		t.Errorf("filter count mismatch")
	}

	t.Logf("MinSum filters: %v", filtersMinSum)
	t.Logf("Entropy filters: %v", filtersEntropy)
}

func TestSelectFilterBigrams(t *testing.T) {
	tests := []struct {
		name   string
		row    []byte
		prev   []byte
		bpp    int
		verify func(*testing.T, FilterType, []byte)
	}{
		{
			name: "returns valid filter type",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{0, 0, 0, 0},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if filterType < FilterNone || filterType > FilterPaeth {
					t.Errorf("filter type %d out of valid range [0-4]", filterType)
				}
			},
		},
		{
			name: "filtered length matches row length",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{0, 0, 0, 0},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 4 {
					t.Errorf("filtered length %d != row length 4", len(filtered))
				}
			},
		},
		{
			name: "deterministic selection",
			row:  []byte{100, 150, 200, 250},
			prev: []byte{50, 100, 150, 200},
			bpp:  1,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				filterType2, filtered2 := SelectFilterBigrams([]byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1)
				if filterType != filterType2 {
					t.Errorf("non-deterministic: first call %d, second call %d", filterType, filterType2)
				}
				if len(filtered) != len(filtered2) {
					t.Errorf("non-deterministic filtered length")
				}
			},
		},
		{
			name: "RGB bpp=3",
			row:  []byte{100, 150, 200, 110, 160, 210},
			prev: []byte{50, 100, 150, 60, 110, 160},
			bpp:  3,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 6 {
					t.Errorf("filtered length %d != row length 6", len(filtered))
				}
			},
		},
		{
			name: "RGBA bpp=4",
			row:  []byte{100, 150, 200, 255, 110, 160, 210, 255},
			prev: []byte{50, 100, 150, 255, 60, 110, 160, 255},
			bpp:  4,
			verify: func(t *testing.T, filterType FilterType, filtered []byte) {
				if len(filtered) != 8 {
					t.Errorf("filtered length %d != row length 8", len(filtered))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filterType, filtered := SelectFilterBigrams(tt.row, tt.prev, tt.bpp)
			tt.verify(t, filterType, filtered)
		})
	}
}

func TestSelectFilterBigramsConsistency(t *testing.T) {
	row := []byte{100, 150, 200, 250, 100, 150, 200, 250}
	prev := []byte{50, 100, 150, 200, 60, 110, 160, 210}
	bpp := 4

	filterType1, filtered1 := SelectFilterBigrams(row, prev, bpp)
	filterType2, filtered2 := SelectFilterWithStrategy(row, prev, bpp, FilterStrategyBigrams)

	if filterType1 != filterType2 {
		t.Errorf("SelectFilterBigrams and SelectFilterWithStrategy(Bigrams) returned different filter types: %d vs %d", filterType1, filterType2)
	}

	if len(filtered1) != len(filtered2) {
		t.Errorf("filtered length mismatch: %d vs %d", len(filtered1), len(filtered2))
	}

	for i := range filtered1 {
		if filtered1[i] != filtered2[i] {
			t.Errorf("filtered data mismatch at index %d: %d vs %d", i, filtered1[i], filtered2[i])
			break
		}
	}
}

func TestSelectAllBigrams(t *testing.T) {
	width, height, bpp := 4, 3, 1
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i * 10)
	}

	filters := SelectAllBigrams(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("SelectAllBigrams returned %d filters, want %d", len(filters), height)
	}

	for i, f := range filters {
		if f < FilterNone || f > FilterPaeth {
			t.Errorf("filter[%d] = %d out of valid range [0-4]", i, f)
		}
	}
}

func TestSelectAllBigramsVsStrategy(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 16)
			pixels[offset+1] = byte(y * 16)
			pixels[offset+2] = byte((x + y) * 8)
			pixels[offset+3] = 255
		}
	}

	filtersDirect := SelectAllBigrams(pixels, width, height, bpp)
	filtersStrategy := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyBigrams)

	if len(filtersDirect) != len(filtersStrategy) {
		t.Errorf("filter count mismatch: %d vs %d", len(filtersDirect), len(filtersStrategy))
	}

	for i := range filtersDirect {
		if filtersDirect[i] != filtersStrategy[i] {
			t.Errorf("filter[%d] mismatch: %d vs %d", i, filtersDirect[i], filtersStrategy[i])
		}
	}
}

func TestBigramsIntegrationWithSelectAll(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"small grayscale", 4, 4, 1},
		{"medium RGB", 8, 8, 3},
		{"large RGBA", 16, 16, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pixels := make([]byte, tc.width*tc.height*tc.bpp)
			for i := range pixels {
				pixels[i] = byte(i % 256)
			}

			filtersBigrams := SelectAllBigrams(pixels, tc.width, tc.height, tc.bpp)
			filtersStrategy := SelectAllWithStrategy(pixels, tc.width, tc.height, tc.bpp, FilterStrategyBigrams)

			if len(filtersBigrams) != tc.height {
				t.Errorf("expected %d filters, got %d", tc.height, len(filtersBigrams))
			}

			if len(filtersStrategy) != tc.height {
				t.Errorf("expected %d strategy filters, got %d", tc.height, len(filtersStrategy))
			}

			for i := 0; i < tc.height; i++ {
				if filtersBigrams[i] != filtersStrategy[i] {
					t.Errorf("row %d: bigrams=%d, strategy=%d", i, filtersBigrams[i], filtersStrategy[i])
				}
			}
		})
	}
}

func TestEarlyTerminationMinSum(t *testing.T) {
	row := []byte{100, 100, 100, 100, 100, 100, 100, 100}
	prev := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	bpp := 1

	scratchNoTerm := NewAdaptiveScratchWithConfig(len(row), bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  false,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		MinFiltersToTry:          DefaultMinFiltersToTry,
	})

	scratchWithTerm := NewAdaptiveScratchWithConfig(len(row), bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  true,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		MinFiltersToTry:          DefaultMinFiltersToTry,
	})

	filterNoTerm, filteredNoTerm := selectMinSumWithScratch(row, prev, bpp, scratchNoTerm)
	filterWithTerm, filteredWithTerm := selectMinSumWithScratch(row, prev, bpp, scratchWithTerm)

	if filterNoTerm != filterWithTerm {
		t.Errorf("early termination changed filter selection: no-term=%d, with-term=%d", filterNoTerm, filterWithTerm)
	}

	for i := range filteredNoTerm {
		if filteredNoTerm[i] != filteredWithTerm[i] {
			t.Errorf("filtered data mismatch at index %d: no-term=%d, with-term=%d", i, filteredNoTerm[i], filteredWithTerm[i])
			break
		}
	}
}

func TestEarlyTerminationEntropy(t *testing.T) {
	row := []byte{100, 150, 200, 250, 100, 150, 200, 250}
	prev := []byte{50, 100, 150, 200, 60, 110, 160, 210}
	bpp := 1

	scratchNoTerm := NewAdaptiveScratchWithConfig(len(row), bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  false,
		FilterSelectionThreshold: DefaultEntropyThreshold,
		MinFiltersToTry:          DefaultMinFiltersToTry,
	})

	scratchWithTerm := NewAdaptiveScratchWithConfig(len(row), bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:   true,
		EarlyTerminationThreshold: 0.01,
		MinFiltersToTry:           5,
	})

	filterNoTerm, filteredNoTerm := selectEntropyWithScratch(row, prev, bpp, scratchNoTerm)
	filterWithTerm, filteredWithTerm := selectEntropyWithScratch(row, prev, bpp, scratchWithTerm)

	if filterNoTerm != filterWithTerm {
		t.Logf("early termination changed filter selection: no-term=%d, with-term=%d", filterNoTerm, filterWithTerm)
	}

	if len(filteredNoTerm) != len(filteredWithTerm) {
		t.Errorf("filtered length mismatch: no-term=%d, with-term=%d", len(filteredNoTerm), len(filteredWithTerm))
	}

	for i := range filteredNoTerm {
		if filteredNoTerm[i] != filteredWithTerm[i] {
			t.Errorf("filtered data mismatch at index %d: no-term=%d, with-term=%d", i, filteredNoTerm[i], filteredWithTerm[i])
			break
		}
	}
}

func TestEarlyTerminationThreshold(t *testing.T) {
	tests := []struct {
		name             string
		row              []byte
		threshold        float64
		expectsEarlyExit bool
	}{
		{
			name:             "low threshold no early exit",
			row:              []byte{100, 150, 200, 250, 100, 150, 200, 250},
			threshold:        0.01,
			expectsEarlyExit: false,
		},
		{
			name:             "high threshold early exit",
			row:              []byte{100, 100, 100, 100, 100, 100, 100, 100},
			threshold:        1000.0,
			expectsEarlyExit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bpp := 1
			prev := make([]byte, len(tt.row))

			scratch := NewAdaptiveScratchWithConfig(len(tt.row), bpp, FilterSelectionConfig{
				EarlyTerminationEnabled:  true,
				FilterSelectionThreshold: tt.threshold,
				MinFiltersToTry:          2,
			})

			filter, _ := selectMinSumWithScratch(tt.row, prev, bpp, scratch)
			if filter < FilterNone || filter > FilterPaeth {
				t.Errorf("invalid filter type: %d", filter)
			}
		})
	}
}

func TestEarlyTerminationMinFiltersToTry(t *testing.T) {
	row := []byte{100, 150, 200, 250, 100, 150, 200, 250}
	prev := make([]byte, len(row))
	bpp := 1

	scratch := NewAdaptiveScratchWithConfig(len(row), bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  true,
		FilterSelectionThreshold: 0.0,
		MinFiltersToTry:          5,
	})

	filter, _ := selectMinSumWithScratch(row, prev, bpp, scratch)
	if filter < FilterNone || filter > FilterPaeth {
		t.Errorf("invalid filter type: %d", filter)
	}
}

func TestEarlyTerminationSpeedImprovement(t *testing.T) {
	width, height, bpp := 256, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	scratchNoTerm := NewAdaptiveScratchWithConfig(width*bpp, bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  false,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		MinFiltersToTry:          DefaultMinFiltersToTry,
	})

	scratchWithTerm := NewAdaptiveScratchWithConfig(width*bpp, bpp, FilterSelectionConfig{
		EarlyTerminationEnabled:  true,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		MinFiltersToTry:          DefaultMinFiltersToTry,
	})

	filtersNoTerm := make([]FilterType, height)
	filtersWithTerm := make([]FilterType, height)
	var prevRow []byte

	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := selectMinSumWithScratch(row, prevRow, bpp, scratchNoTerm)
		filtersNoTerm[y] = filterType
		prevRow = row
	}

	prevRow = nil
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]
		filterType, _ := selectMinSumWithScratch(row, prevRow, bpp, scratchWithTerm)
		filtersWithTerm[y] = filterType
		prevRow = row
	}

	mismatchCount := 0
	for i := range filtersNoTerm {
		if filtersNoTerm[i] != filtersWithTerm[i] {
			mismatchCount++
		}
	}

	if mismatchCount > height/10 {
		t.Errorf("too many filter mismatches: %d out of %d (%.1f%%)", mismatchCount, height, float64(mismatchCount)*100/float64(height))
	}
}

func TestEarlyTerminationNoRegression(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"small grayscale", 4, 4, 1},
		{"medium RGB", 32, 32, 3},
		{"large RGBA", 64, 64, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pixels := make([]byte, tc.width*tc.height*tc.bpp)
			for i := range pixels {
				pixels[i] = byte(i * 7 % 256)
			}

			scratchNoTerm := NewAdaptiveScratchWithConfig(tc.width*tc.bpp, tc.bpp, FilterSelectionConfig{
				EarlyTerminationEnabled:  false,
				FilterSelectionThreshold: DefaultMinSumThreshold,
				MinFiltersToTry:          DefaultMinFiltersToTry,
			})

			scratchWithTerm := NewAdaptiveScratchWithConfig(tc.width*tc.bpp, tc.bpp, FilterSelectionConfig{
				EarlyTerminationEnabled:  true,
				FilterSelectionThreshold: DefaultMinSumThreshold,
				MinFiltersToTry:          DefaultMinFiltersToTry,
			})

			filtersNoTerm := SelectAllWithStrategyAndScratch(pixels, tc.width, tc.height, tc.bpp, FilterStrategyMinSum, scratchNoTerm)
			filtersWithTerm := SelectAllWithStrategyAndScratch(pixels, tc.width, tc.height, tc.bpp, FilterStrategyMinSum, scratchWithTerm)

			if len(filtersNoTerm) != tc.height || len(filtersWithTerm) != tc.height {
				t.Errorf("filter count mismatch")
			}

			for i := range filtersNoTerm {
				if filtersNoTerm[i] != filtersWithTerm[i] {
					t.Errorf("row %d: no-term=%d, with-term=%d", i, filtersNoTerm[i], filtersWithTerm[i])
				}
			}
		})
	}
}

