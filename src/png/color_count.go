package png

import "sort"

// colorKey is used as a map key for color counting.
type colorKey struct {
	r, g, b uint8
}

// CountColors counts the frequency of each unique color in the pixel data.
// colorType: 2=RGB, 6=RGBA
// Returns a map of Color to count, sorted by count descending.
func CountColors(pixels []byte, colorType int) map[Color]int {
	colorMap := make(map[Color]int)

	bpp := BytesPerPixel(ColorType(colorType))

	for i := 0; i < len(pixels); i += bpp {
		c := Color{
			R: pixels[i],
			G: pixels[i+1],
			B: pixels[i+2],
		}
		colorMap[c]++
	}

	return colorMap
}

// CountColorsWithAlpha counts colors including alpha information.
func CountColorsWithAlpha(pixels []byte, colorType int) map[ColorWithCount]int {
	colorMap := make(map[ColorWithCount]int)

	bpp := BytesPerPixel(ColorType(colorType))

	for i := 0; i < len(pixels); i += bpp {
		cwc := ColorWithCount{
			Color: Color{
				R: pixels[i],
				G: pixels[i+1],
				B: pixels[i+2],
			},
			Count: 1,
		}
		colorMap[cwc]++
	}

	return colorMap
}

// ToColorWithCountSlice converts a color count map to a sorted slice.
func ToColorWithCountSlice(colorMap map[Color]int) []ColorWithCount {
	result := make([]ColorWithCount, 0, len(colorMap))

	for c, count := range colorMap {
		result = append(result, ColorWithCount{
			Color: c,
			Count: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

// UniqueColorCount returns the number of unique colors in the pixel data.
func UniqueColorCount(pixels []byte, colorType int) int {
	colorMap := make(map[colorKey]struct{})

	bpp := BytesPerPixel(ColorType(colorType))

	for i := 0; i < len(pixels); i += bpp {
		key := colorKey{
			r: pixels[i],
			g: pixels[i+1],
			b: pixels[i+2],
		}
		colorMap[key] = struct{}{}
	}

	return len(colorMap)
}
