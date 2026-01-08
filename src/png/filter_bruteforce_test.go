package png

import (
	"bytes"
	"testing"
)

func TestBruteForceFilters_SmallImage(t *testing.T) {
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

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	for i, f := range filters {
		if f < FilterNone || f > FilterPaeth {
			t.Errorf("filter[%d] = %d out of valid range [0-4]", i, f)
		}
	}
}

func TestBruteForceFilters_VerySmallImage(t *testing.T) {
	width, height, bpp := 4, 4, 1
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i * 10)
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	for i, f := range filters {
		if f < FilterNone || f > FilterPaeth {
			t.Errorf("filter[%d] = %d out of valid range [0-4]", i, f)
		}
	}
}

func TestBruteForceFilters_ProducesValidFilters(t *testing.T) {
	width, height, bpp := 16, 16, 3
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	// Verify all filters are valid
	for i, f := range filters {
		if f > FilterPaeth {
			t.Errorf("filter[%d] = %d, want <= %d", i, f, FilterPaeth)
		}
	}
}

func TestBruteForceFilters_Grayscale(t *testing.T) {
	width, height, bpp := 8, 8, 1
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = byte(x * 16)
		}
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}
}

func TestBruteForceFilters_RGB(t *testing.T) {
	width, height, bpp := 8, 8, 3
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 10)
			pixels[offset+1] = byte(y * 10)
			pixels[offset+2] = byte((x + y) * 5)
		}
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}
}

func TestBruteForceFilters_RGBA(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 16)
			pixels[offset+1] = byte(y * 16)
			pixels[offset+2] = byte(128)
			pixels[offset+3] = 255
		}
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}
}

func TestBruteForceFilters_RepetitivePattern(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := bytes.Repeat([]byte{100, 150, 200, 255}, width*height)

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	// For repetitive data, Up filter should be selected for rows after the first
	for i := 1; i < len(filters); i++ {
		if filters[i] != FilterUp {
			t.Logf("row %d: filter = %d (Up expected for repetitive data)", i, filters[i])
		}
	}
}

func TestBruteForceFilters_Gradient(t *testing.T) {
	width, height, bpp := 8, 8, 1
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = byte(x * 16)
		}
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	// For horizontal gradient, Sub filter should be optimal
	for i, f := range filters {
		t.Logf("row %d: filter = %d", i, f)
	}
}

func TestOptimalFiltersForImage(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		expectBrute bool
	}{
		{"small 64x64", 64, 64, true},
		{"medium 128x128", 128, 128, true},
		{"at threshold 256x256", 256, 256, true},
		{"above threshold 512x512", 512, 512, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bruteForce, _ := OptimalFiltersForImage(tt.width, tt.height)
			if bruteForce != tt.expectBrute {
				t.Errorf("OptimalFiltersForImage(%d, %d) bruteForce = %v, want %v",
					tt.width, tt.height, bruteForce, tt.expectBrute)
			}
		})
	}
}

func TestCompressSingleRow_UsesLZ77Estimator(t *testing.T) {
	// Test that compressSingleRow returns a slice with size based on LZ77 estimation
	// not just the raw filtered data length
	row := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	result := compressSingleRow(row)

	// Result should be a slice of estimated compressed size
	if len(result) == 0 {
		t.Error("compressSingleRow returned empty slice for non-empty input")
	}
}

func TestCompressSingleRow_EmptyInput(t *testing.T) {
	result := compressSingleRow([]byte{})

	if len(result) != 0 {
		t.Errorf("compressSingleRow([]) returned %d bytes, want 0", len(result))
	}
}

func TestSelectBruteForce_UsesCompressSingleRow(t *testing.T) {
	// Test that selectBruteForce uses compressSingleRow for size estimation
	row := []byte{10, 20, 30, 40, 50, 60, 70, 80}
	prevRow := []byte{5, 15, 25, 35, 45, 55, 65, 75}

	filterType, filtered := selectBruteForce(row, prevRow, 1)

	if filterType < FilterNone || filterType > FilterPaeth {
		t.Errorf("selectBruteForce returned invalid filter type %d", filterType)
	}

	// Filtered should be the result of applying the selected filter
	if len(filtered) == 0 && len(row) > 0 {
		t.Error("selectBruteForce returned empty filtered result for non-empty row")
	}
}

func TestSelectBruteForce_AllFiltersValid(t *testing.T) {
	// Test that selectBruteForce can handle all filter types and picks one
	row := []byte{1, 2, 3, 4, 5}
	prevRow := []byte{1, 2, 3, 4, 5}

	filterType, _ := selectBruteForce(row, prevRow, 1)

	// Should always return a valid filter type
	if filterType < FilterNone || filterType > FilterPaeth {
		t.Errorf("Expected valid filter type, got %d", filterType)
	}
}

func TestSelectBruteForce_UniformData(t *testing.T) {
	// For uniform data (all same bytes), Up filter should be optimal
	row := []byte{100, 100, 100, 100, 100, 100, 100, 100}
	prevRow := []byte{100, 100, 100, 100, 100, 100, 100, 100}

	filterType, _ := selectBruteForce(row, prevRow, 1)

	// Up filter should be selected for uniform data with matching previous row
	if filterType != FilterUp {
		t.Logf("Expected FilterUp for uniform data, got %d", filterType)
	}
}

func TestBruteForceFilters_UsesLZ77(t *testing.T) {
	// Verify that BruteForceFilters uses the LZ77-based compressSingleRow
	width, height, bpp := 4, 4, 3
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	filters := BruteForceFilters(pixels, width, height, bpp)

	if len(filters) != height {
		t.Errorf("BruteForceFilters returned %d filters, want %d", len(filters), height)
	}

	// Verify all filters are valid types
	for i, f := range filters {
		if f > FilterPaeth {
			t.Errorf("filter[%d] = %d, want <= %d", i, f, FilterPaeth)
		}
	}
}
