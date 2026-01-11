package png

import (
	"math"
	"sort"
)

type ColorCount struct {
	Color Color
	Count int
	R     float64
	G     float64
	B     float64
}

func ToColorCount(colorMap map[Color]int) []ColorCount {
	result := make([]ColorCount, 0, len(colorMap))

	for c, count := range colorMap {
		result = append(result, ColorCount{
			Color: c,
			Count: count,
			R:     float64(c.R),
			G:     float64(c.G),
			B:     float64(c.B),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func colorDistance(c1, c2 Color) float64 {
	dr := float64(c1.R) - float64(c2.R)
	dg := float64(c1.G) - float64(c2.G)
	db := float64(c1.B) - float64(c2.B)
	return dr*dr + dg*dg + db*db
}

func RefinePaletteKmeans(palette *Palette, colors []ColorCount, iterations int) []Color {
	if len(colors) == 0 || palette.NumColors == 0 {
		return palette.Colors[:palette.NumColors]
	}

	if len(colors) == 1 && palette.NumColors > 0 {
		return palette.Colors[:palette.NumColors]
	}

	numColors := palette.NumColors
	result := make([]Color, numColors)
	copy(result, palette.Colors[:numColors])

	for iter := 0; iter < iterations; iter++ {
		clusterSums := make([]float64, numColors*3)
		clusterCounts := make([]int, numColors)

		for _, c := range colors {
			bestIdx := 0
			bestDist := colorDistance(c.Color, result[0])
			for i := 1; i < numColors; i++ {
				dist := colorDistance(c.Color, result[i])
				if dist < bestDist {
					bestDist = dist
					bestIdx = i
				}
			}
			clusterSums[bestIdx*3+0] += c.R * float64(c.Count)
			clusterSums[bestIdx*3+1] += c.G * float64(c.Count)
			clusterSums[bestIdx*3+2] += c.B * float64(c.Count)
			clusterCounts[bestIdx] += c.Count
		}

		changed := false
		for i := 0; i < numColors; i++ {
			if clusterCounts[i] == 0 {
				continue
			}

			newColor := Color{
				R: uint8(math.Round(clusterSums[i*3+0] / float64(clusterCounts[i]))),
				G: uint8(math.Round(clusterSums[i*3+1] / float64(clusterCounts[i]))),
				B: uint8(math.Round(clusterSums[i*3+2] / float64(clusterCounts[i]))),
			}

			if newColor.R != result[i].R || newColor.G != result[i].G || newColor.B != result[i].B {
				result[i] = newColor
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	return result
}
