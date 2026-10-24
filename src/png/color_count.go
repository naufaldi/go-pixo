package png

import "sort"

// colorKey is used as a map key for color counting.
type colorKey struct {
	r, g, b uint8
}

// CountColors counts the frequency of each unique color in the pixel data.
// colorType: 2=RGB, 6=RGBA
// Returns an unordered map of Color to count.
func CountColors(pixels []byte, colorType int) map[Color]int {
	return countColorsWithProgress(pixels, colorType, nil)
}

func countColorsWithProgress(pixels []byte, colorType int, callback func(string, int)) map[Color]int {
	colorMap := make(map[Color]int)

	bpp := BytesPerPixel(ColorType(colorType))

	progress := newWorkProgress(callback, "histogram", len(pixels)/bpp, 4096)
	for i := 0; i < len(pixels); i += bpp {
		c := Color{
			R: pixels[i],
			G: pixels[i+1],
			B: pixels[i+2],
		}
		colorMap[c]++
		if completed := i/bpp + 1; completed >= progress.next {
			progress.report(completed)
		}
	}

	progress.finish()
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

// ToColorWithCountSlice sorts by count descending, then RGB ascending for ties.
func ToColorWithCountSlice(colorMap map[Color]int) []ColorWithCount {
	result := make([]ColorWithCount, 0, len(colorMap))

	for c, count := range colorMap {
		result = append(result, ColorWithCount{
			Color: c,
			Count: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		// Median cut must receive the same order regardless of map iteration.
		if a.R != b.R {
			return a.R < b.R
		}
		if a.G != b.G {
			return a.G < b.G
		}
		return a.B < b.B
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
