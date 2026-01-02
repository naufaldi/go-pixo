package png

import "testing"

func TestFilterReconstructRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		row      []byte
		prev     []byte
		bpp      int
		filterFn func([]byte, []byte, int) []byte
		reconFn  func([]byte, []byte, int) []byte
	}{
		{
			name:     "None",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{0, 0, 0, 0},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterNone(r) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructNone(f) },
		},
		{
			name:     "Sub",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{0, 0, 0, 0},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterSub(r, b) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructSub(f, b) },
		},
		{
			name:     "Up",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{50, 100, 150, 200},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterUp(r, p) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructUp(f, p) },
		},
		{
			name:     "Average",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{50, 100, 150, 200},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterAverage(r, p, b) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructAverage(f, p, b) },
		},
		{
			name:     "Paeth",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{50, 100, 150, 200},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterPaeth(r, p, b) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructPaeth(f, p, b) },
		},
		{
			name:     "Sub RGB",
			row:      []byte{100, 150, 200, 110, 160, 210},
			prev:     []byte{0, 0, 0, 0, 0, 0},
			bpp:      3,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterSub(r, b) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructSub(f, b) },
		},
		{
			name:     "Up RGBA",
			row:      []byte{100, 150, 200, 255, 110, 160, 210, 255},
			prev:     []byte{50, 100, 150, 255, 60, 110, 160, 255},
			bpp:      4,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterUp(r, p) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructUp(f, p) },
		},
		{
			name:     "First row (prev all zeros)",
			row:      []byte{100, 150, 200, 250},
			prev:     []byte{0, 0, 0, 0},
			bpp:      1,
			filterFn: func(r []byte, p []byte, b int) []byte { return ApplyFilterAverage(r, p, b) },
			reconFn:  func(f []byte, p []byte, b int) []byte { return ReconstructAverage(f, p, b) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := tt.filterFn(tt.row, tt.prev, tt.bpp)
			reconstructed := tt.reconFn(filtered, tt.prev, tt.bpp)

			if len(reconstructed) != len(tt.row) {
				t.Fatalf("reconstructed length %d != original length %d",
					len(reconstructed), len(tt.row))
			}

			for i := range tt.row {
				if reconstructed[i] != tt.row[i] {
					t.Errorf("position %d: reconstructed %d != original %d",
						i, reconstructed[i], tt.row[i])
				}
			}
		})
	}
}
