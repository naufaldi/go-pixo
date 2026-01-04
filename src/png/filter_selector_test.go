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
