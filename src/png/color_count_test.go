package png

import (
	"reflect"
	"testing"
)

func TestCountColors(t *testing.T) {
	tests := []struct {
		name      string
		pixels    []byte
		colorType int
		wantLen   int
	}{
		{
			name:      "single color",
			pixels:    []byte{255, 0, 0, 255, 0, 0, 255, 0, 0},
			colorType: 2,
			wantLen:   1,
		},
		{
			name:      "three colors",
			pixels:    []byte{255, 0, 0, 0, 255, 0, 0, 0, 255},
			colorType: 2,
			wantLen:   3,
		},
		{
			name:      "two colors mixed",
			pixels:    []byte{255, 0, 0, 0, 255, 0, 255, 0, 0, 0, 255, 0},
			colorType: 2,
			wantLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountColors(tt.pixels, tt.colorType)
			if len(got) != tt.wantLen {
				t.Errorf("CountColors() = %v colors, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestCountColorsWithFrequency(t *testing.T) {
	pixels := []byte{255, 0, 0, 255, 0, 0, 0, 255, 0}
	got := CountColors(pixels, 2)

	// Red should appear 2 times
	if got[Color{255, 0, 0}] != 2 {
		t.Errorf("CountColors() red count = %v, want 2", got[Color{255, 0, 0}])
	}

	// Green should appear 1 time
	if got[Color{0, 255, 0}] != 1 {
		t.Errorf("CountColors() green count = %v, want 1", got[Color{0, 255, 0}])
	}

	// Blue should appear 0 times
	if got[Color{0, 0, 255}] != 0 {
		t.Errorf("CountColors() blue count = %v, want 0", got[Color{0, 0, 255}])
	}
}

func TestCountColorsRGBA(t *testing.T) {
	pixels := []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255}
	got := CountColors(pixels, 6)

	// RGB is used for counting, alpha ignored
	if len(got) != 3 {
		t.Errorf("CountColors(RGBA) = %v colors, want 3", len(got))
	}
}

func TestToColorWithCountSlice(t *testing.T) {
	colorMap := map[Color]int{
		{255, 0, 0}: 5,
		{0, 255, 0}: 3,
		{0, 0, 255}: 1,
	}

	slice := ToColorWithCountSlice(colorMap)

	if len(slice) != 3 {
		t.Errorf("ToColorWithCountSlice() = %v, want 3", len(slice))
	}

	// Should be sorted by count descending
	if slice[0].Count != 5 {
		t.Errorf("ToColorWithCountSlice() first count = %v, want 5", slice[0].Count)
	}
	if slice[1].Count != 3 {
		t.Errorf("ToColorWithCountSlice() second count = %v, want 3", slice[1].Count)
	}
	if slice[2].Count != 1 {
		t.Errorf("ToColorWithCountSlice() third count = %v, want 1", slice[2].Count)
	}
}

func TestUniqueColorCount(t *testing.T) {
	tests := []struct {
		name      string
		pixels    []byte
		colorType int
		want      int
	}{
		{
			name:      "all same",
			pixels:    []byte{100, 100, 100, 100, 100, 100, 100, 100, 100},
			colorType: 2,
			want:      1,
		},
		{
			name:      "all different",
			pixels:    []byte{0, 0, 0, 1, 1, 1, 2, 2, 2},
			colorType: 2,
			want:      3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UniqueColorCount(tt.pixels, tt.colorType)
			if got != tt.want {
				t.Errorf("UniqueColorCount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountColorsEmpty(t *testing.T) {
	got := CountColors([]byte{}, 2)
	if len(got) != 0 {
		t.Errorf("CountColors() on empty = %v, want 0", len(got))
	}
}

func TestColorWithCountStruct(t *testing.T) {
	cwc := ColorWithCount{
		Color: Color{255, 0, 0},
		Count: 5,
	}

	if cwc.R != 255 {
		t.Errorf("ColorWithCount.R = %v, want 255", cwc.R)
	}
	if cwc.Count != 5 {
		t.Errorf("ColorWithCount.Count = %v, want 5", cwc.Count)
	}
}

func TestColorEquality(t *testing.T) {
	c1 := Color{255, 128, 64}
	c2 := Color{255, 128, 64}
	c3 := Color{255, 128, 63}

	if c1 != c2 {
		t.Errorf("Equal colors should be equal")
	}

	if c1 == c3 {
		t.Errorf("Different colors should not be equal")
	}
}

func TestColorMapLookup(t *testing.T) {
	colorMap := make(map[Color]int)
	colorMap[Color{255, 0, 0}] = 5

	if colorMap[Color{255, 0, 0}] != 5 {
		t.Errorf("Color map lookup failed")
	}

	if colorMap[Color{0, 255, 0}] != 0 {
		t.Errorf("Color map missing key should return 0")
	}
}

func TestToColorWithCountSliceEmpty(t *testing.T) {
	slice := ToColorWithCountSlice(map[Color]int{})
	if len(slice) != 0 {
		t.Errorf("ToColorWithCountSlice() on empty = %v, want 0", len(slice))
	}
}

func TestToColorWithCountSliceSorted(t *testing.T) {
	colorMap := map[Color]int{
		{255, 0, 0}: 1,
		{0, 255, 0}: 10,
		{0, 0, 255}: 5,
	}

	slice := ToColorWithCountSlice(colorMap)

	// Check that it's sorted by count descending
	if slice[0].Count != 10 {
		t.Errorf("Slice should be sorted by count descending, first = %v", slice[0].Count)
	}
	if slice[1].Count != 5 {
		t.Errorf("Slice should be sorted by count descending, second = %v", slice[1].Count)
	}
	if slice[2].Count != 1 {
		t.Errorf("Slice should be sorted by count descending, third = %v", slice[2].Count)
	}
}

func TestUniqueColorCountEmpty(t *testing.T) {
	got := UniqueColorCount([]byte{}, 2)
	if got != 0 {
		t.Errorf("UniqueColorCount() on empty = %v, want 0", got)
	}
}

func TestCountColorsWithAlphaType(t *testing.T) {
	// 6 = RGBA color type
	pixels := []byte{255, 0, 0, 255, 0, 255, 0, 255}
	got := CountColors(pixels, 6)

	// Should still count by RGB, ignoring alpha
	if len(got) != 2 {
		t.Errorf("CountColors() with RGBA = %v colors, want 2", len(got))
	}
}

func TestReflectTypeForColorMap(t *testing.T) {
	colorMap := make(map[Color]int)
	mapType := reflect.TypeOf(colorMap)
	keyType := mapType.Key()

	if keyType.Kind() != reflect.Struct {
		t.Errorf("Color map key should be struct, got %v", keyType.Kind())
	}
}
