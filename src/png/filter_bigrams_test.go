package png

import (
	"bytes"
	"testing"

	"github.com/mac/go-pixo/src/compress"
)

func TestCountDistinctBigrams(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{
			name:     "empty data",
			data:     []byte{},
			expected: 0,
		},
		{
			name:     "single byte",
			data:     []byte{100},
			expected: 0,
		},
		{
			name:     "two identical bytes",
			data:     []byte{100, 100},
			expected: 1,
		},
		{
			name:     "two different bytes",
			data:     []byte{100, 200},
			expected: 1,
		},
		{
			name:     "three bytes all same",
			data:     []byte{100, 100, 100},
			expected: 1,
		},
		{
			name:     "three bytes with one pair different",
			data:     []byte{100, 100, 200},
			expected: 2,
		},
		{
			name:     "all unique pairs",
			data:     []byte{0, 1, 2, 3},
			expected: 3,
		},
		{
			name:     "repeating pattern",
			data:     []byte{100, 150, 100, 150, 100, 150},
			expected: 2,
		},
		{
			name:     "sequential bytes",
			data:     []byte{1, 2, 3, 4, 5, 6, 7, 8},
			expected: 7,
		},
		{
			name:     "all zeros",
			data:     []byte{0, 0, 0, 0, 0, 0},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countDistinctBigrams(tt.data)
			if got != tt.expected {
				t.Errorf("countDistinctBigrams(%v) = %d, want %d", tt.data, got, tt.expected)
			}
		})
	}
}

func TestSelectBigrams(t *testing.T) {
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
				filterType2, filtered2 := selectBigrams([]byte{100, 150, 200, 250}, []byte{50, 100, 150, 200}, 1)
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
			filterType, filtered := selectBigrams(tt.row, tt.prev, tt.bpp)
			tt.verify(t, filterType, filtered)
		})
	}
}

func TestApplyBigrams(t *testing.T) {
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
			filtered := ApplyBigrams(tt.row, tt.prev, tt.bpp)
			if len(filtered) != len(tt.row) {
				t.Errorf("filtered length %d != row length %d", len(filtered), len(tt.row))
			}
		})
	}
}

func TestSelectFilterWithStrategyBigrams(t *testing.T) {
	row := []byte{100, 150, 200, 250, 100, 150, 200, 250}
	prev := []byte{50, 100, 150, 200, 60, 110, 160, 210}
	bpp := 4

	filterType, filtered := SelectFilterWithStrategy(row, prev, bpp, FilterStrategyBigrams)

	if filterType < FilterNone || filterType > FilterPaeth {
		t.Errorf("filter type %d out of valid range [0-4]", filterType)
	}
	if len(filtered) != len(row) {
		t.Errorf("filtered length %d != row length %d", len(filtered), len(row))
	}
}

func TestBigramsFilterStrategyVsMinSum(t *testing.T) {
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
	filtersBigrams := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyBigrams)

	if len(filtersMinSum) != height || len(filtersBigrams) != height {
		t.Errorf("filter count mismatch")
	}

	t.Logf("MinSum filters: %v", filtersMinSum)
	t.Logf("Bigrams filters: %v", filtersBigrams)
}

func TestBigramsCompressionRatio(t *testing.T) {
	width, height, bpp := 16, 16, 4
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

	var minSumFiltered bytes.Buffer
	var bigramsFiltered bytes.Buffer

	var prevRow []byte
	for y := 0; y < height; y++ {
		offset := y * width * bpp
		row := pixels[offset : offset+width*bpp]

		_, filteredMinSum := SelectFilterWithStrategy(row, prevRow, bpp, FilterStrategyMinSum)
		minSumFiltered.Write(filteredMinSum)

		_, filteredBigrams := SelectFilterWithStrategy(row, prevRow, bpp, FilterStrategyBigrams)
		bigramsFiltered.Write(filteredBigrams)

		prevRow = row
	}

	minSumSize := compressSize(minSumFiltered.Bytes())
	bigramsSize := compressSize(bigramsFiltered.Bytes())

	t.Logf("MinSum filtered size: %d bytes", minSumSize)
	t.Logf("Bigrams filtered size: %d bytes", bigramsSize)
	t.Logf("Bigrams vs MinSum ratio: %.2f%%", float64(bigramsSize)/float64(minSumSize)*100)

	if bigramsSize > minSumSize {
		t.Logf("Note: Bigrams strategy produced larger output (%.2f%% larger)",
			float64(bigramsSize-minSumSize)/float64(minSumSize)*100)
	}
}

func compressSize(data []byte) int {
	size, _ := compress.EstimateCompressedSize(data)
	return size
}
