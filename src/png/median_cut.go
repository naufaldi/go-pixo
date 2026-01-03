package png

import "sort"

// bucket represents a collection of colors for median cut.
type bucket struct {
	colors []ColorWithCount
}

// MedianCut performs median cut color quantization.
// It recursively splits the color space until the target number of colors is reached.
func MedianCut(colorsWithCount []ColorWithCount, maxColors int) []Color {
	if len(colorsWithCount) == 0 {
		return []Color{}
	}

	if len(colorsWithCount) <= maxColors {
		result := make([]Color, len(colorsWithCount))
		for i, cwc := range colorsWithCount {
			result[i] = cwc.Color
		}
		return result
	}

	buckets := []bucket{{colors: colorsWithCount}}

	for len(buckets) < maxColors {
		// Find largest bucket
		largestIdx := -1
		maxSize := 0
		for i := range buckets {
			if len(buckets[i].colors) > maxSize {
				maxSize = len(buckets[i].colors)
				largestIdx = i
			}
		}

		if largestIdx == -1 || maxSize < 2 {
			break
		}

		// Split the largest bucket
		left, right := splitBucket(buckets[largestIdx].colors)

		// Replace the largest bucket with left, add right
		buckets[largestIdx].colors = left
		if len(right) > 0 {
			buckets = append(buckets, bucket{colors: right})
		}
	}

	result := make([]Color, 0, maxColors)
	for _, b := range buckets {
		if len(b.colors) > 0 {
			result = append(result, averageColors(b.colors))
		}
	}

	return result
}

// splitBucket splits a bucket into two at the median.
func splitBucket(colors []ColorWithCount) ([]ColorWithCount, []ColorWithCount) {
	if len(colors) < 2 {
		return colors, nil
	}

	minR, maxR := uint8(255), uint8(0)
	minG, maxG := uint8(255), uint8(0)
	minB, maxB := uint8(255), uint8(0)

	for _, c := range colors {
		if c.R < minR {
			minR = c.R
		}
		if c.R > maxR {
			maxR = c.R
		}
		if c.G < minG {
			minG = c.G
		}
		if c.G > maxG {
			maxG = c.G
		}
		if c.B < minB {
			minB = c.B
		}
		if c.B > maxB {
			maxB = c.B
		}
	}

	rangeR := int(maxR) - int(minR)
	rangeG := int(maxG) - int(minG)
	rangeB := int(maxB) - int(minB)

	sortBy := 0
	maxRange := rangeR
	if rangeG > maxRange {
		maxRange = rangeG
		sortBy = 1
	}
	if rangeB > maxRange {
		maxRange = rangeB
		sortBy = 2
	}

	sorted := make([]ColorWithCount, len(colors))
	copy(sorted, colors)

	sort.Slice(sorted, func(i, j int) bool {
		switch sortBy {
		case 0:
			return sorted[i].R < sorted[j].R
		case 1:
			return sorted[i].G < sorted[j].G
		default:
			return sorted[i].B < sorted[j].B
		}
	})

	mid := len(sorted) / 2

	return sorted[:mid], sorted[mid:]
}

// averageColors calculates the average color of all colors in the bucket.
func averageColors(colors []ColorWithCount) Color {
	var totalR, totalG, totalB int
	var totalCount int

	for _, c := range colors {
		totalR += int(c.Color.R) * c.Count
		totalG += int(c.Color.G) * c.Count
		totalB += int(c.Color.B) * c.Count
		totalCount += c.Count
	}

	if totalCount == 0 {
		totalCount = len(colors)
	}

	return Color{
		R: uint8(totalR / totalCount),
		G: uint8(totalG / totalCount),
		B: uint8(totalB / totalCount),
	}
}

// MedianCutWithAlpha performs median cut including alpha channel.
func MedianCutWithAlpha(colorsWithCount []ColorWithCount, maxColors int) []Color {
	if len(colorsWithCount) == 0 {
		return []Color{}
	}

	if len(colorsWithCount) <= maxColors {
		result := make([]Color, len(colorsWithCount))
		for i, cwc := range colorsWithCount {
			result[i] = cwc.Color
		}
		return result
	}

	buckets := []bucket{{colors: colorsWithCount}}

	for len(buckets) < maxColors {
		largestIdx := -1
		maxSize := 0
		for i := range buckets {
			if len(buckets[i].colors) > maxSize {
				maxSize = len(buckets[i].colors)
				largestIdx = i
			}
		}

		if largestIdx == -1 || maxSize < 2 {
			break
		}

		left, right := splitBucket(buckets[largestIdx].colors)

		buckets[largestIdx].colors = left
		if len(right) > 0 {
			buckets = append(buckets, bucket{colors: right})
		}
	}

	result := make([]Color, 0, maxColors)
	for _, b := range buckets {
		if len(b.colors) > 0 {
			result = append(result, averageColors(b.colors))
		}
	}

	return result
}
