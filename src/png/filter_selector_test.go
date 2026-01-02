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
