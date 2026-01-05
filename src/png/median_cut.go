package png

import (
	"math"
	"sort"
)

// bucket represents a collection of colors for median cut.
type bucket struct {
	colors []ColorWithCount
}

// MedianCut performs median cut color quantization.
// It recursively splits the color space until the target number of colors is reached.
func MedianCut(colorsWithCount []ColorWithCount, maxColors int) []Color {
	return MedianCutWithQuality(colorsWithCount, maxColors, 1.0)
}

// MedianCutWithQuality performs median cut with configurable quality.
// The quality parameter (0.0-1.0) controls the trade-off between color accuracy and palette size.
// Lower quality values result in more aggressive quantization with fewer distinct colors.
func MedianCutWithQuality(colorsWithCount []ColorWithCount, maxColors int, quality float64) []Color {
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

	// Adjust effective max colors based on quality
	effectiveMaxColors := maxColors
	if quality < 1.0 {
		effectiveMaxColors = int(float64(maxColors) * (0.5 + quality*0.5))
		if effectiveMaxColors < 2 {
			effectiveMaxColors = 2
		}
	}

	buckets := []bucket{{colors: colorsWithCount}}

	for len(buckets) < effectiveMaxColors {
		// Find bucket with most color variance
		largestIdx := -1
		maxVariance := 0.0
		for i := range buckets {
			variance := calculateColorVariance(buckets[i].colors)
			if variance > maxVariance && len(buckets[i].colors) >= 2 {
				maxVariance = variance
				largestIdx = i
			}
		}

		if largestIdx == -1 || len(buckets[largestIdx].colors) < 2 {
			break
		}

		// Split the bucket with highest variance
		left, right := splitBucketWithQuality(buckets[largestIdx].colors, quality)

		// Replace the bucket with left, add right
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

// calculateColorVariance calculates the color variance within a bucket.
// Used to prioritize splitting buckets with the most color spread.
func calculateColorVariance(colors []ColorWithCount) float64 {
	if len(colors) == 0 {
		return 0
	}

	var totalR, totalG, totalB int
	totalCount := 0
	for _, c := range colors {
		totalR += int(c.Color.R) * c.Count
		totalG += int(c.Color.G) * c.Count
		totalB += int(c.Color.B) * c.Count
		totalCount += c.Count
	}

	meanR := float64(totalR) / float64(totalCount)
	meanG := float64(totalG) / float64(totalCount)
	meanB := float64(totalB) / float64(totalCount)

	var variance float64
	for _, c := range colors {
		weight := float64(c.Count)
		dr := float64(c.Color.R) - meanR
		dg := float64(c.Color.G) - meanG
		db := float64(c.Color.B) - meanB
		variance += weight * (dr*dr + dg*dg + db*db)
	}

	return variance / float64(totalCount)
}

// splitBucketWithQuality splits a bucket at the median with quality consideration.
// Higher quality values result in more balanced splits.
func splitBucketWithQuality(colors []ColorWithCount, quality float64) ([]ColorWithCount, []ColorWithCount) {
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

	// Adjust split point based on quality
	splitRatio := 0.5
	if quality < 0.5 {
		splitRatio = 0.5 - (0.5-quality)*0.3
	} else if quality > 0.5 {
		splitRatio = 0.5 + (quality-0.5)*0.3
	}

	mid := int(float64(len(sorted)) * splitRatio)
	if mid < 1 {
		mid = 1
	}
	if mid >= len(sorted) {
		mid = len(sorted) - 1
	}

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
	return MedianCutWithAlphaAndQuality(colorsWithCount, maxColors, 1.0)
}

// MedianCutWithAlphaAndQuality performs median cut with alpha channel and quality control.
func MedianCutWithAlphaAndQuality(colorsWithCount []ColorWithCount, maxColors int, quality float64) []Color {
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

	// Adjust effective max colors based on quality
	effectiveMaxColors := maxColors
	if quality < 1.0 {
		effectiveMaxColors = int(float64(maxColors) * (0.5 + quality*0.5))
		if effectiveMaxColors < 2 {
			effectiveMaxColors = 2
		}
	}

	buckets := []bucket{{colors: colorsWithCount}}

	for len(buckets) < effectiveMaxColors {
		largestIdx := -1
		maxVariance := 0.0
		for i := range buckets {
			variance := calculateColorVarianceWithAlpha(buckets[i].colors)
			if variance > maxVariance && len(buckets[i].colors) >= 2 {
				maxVariance = variance
				largestIdx = i
			}
		}

		if largestIdx == -1 || len(buckets[largestIdx].colors) < 2 {
			break
		}

		left, right := splitBucketWithQualityAndAlpha(buckets[largestIdx].colors, quality)

		buckets[largestIdx].colors = left
		if len(right) > 0 {
			buckets = append(buckets, bucket{colors: right})
		}
	}

	result := make([]Color, 0, maxColors)
	for _, b := range buckets {
		if len(b.colors) > 0 {
			result = append(result, averageColorsWithAlpha(b.colors))
		}
	}

	return result
}

// calculateColorVarianceWithAlpha calculates color variance including alpha channel.
func calculateColorVarianceWithAlpha(colors []ColorWithCount) float64 {
	if len(colors) == 0 {
		return 0
	}

	var totalR, totalG, totalB int
	totalCount := 0
	for _, c := range colors {
		totalR += int(c.Color.R) * c.Count
		totalG += int(c.Color.G) * c.Count
		totalB += int(c.Color.B) * c.Count
		totalCount += c.Count
	}

	meanR := float64(totalR) / float64(totalCount)
	meanG := float64(totalG) / float64(totalCount)
	meanB := float64(totalB) / float64(totalCount)

	var variance float64
	for _, c := range colors {
		weight := float64(c.Count)
		dr := float64(c.Color.R) - meanR
		dg := float64(c.Color.G) - meanG
		db := float64(c.Color.B) - meanB
		variance += weight * (dr*dr + dg*dg + db*db)
	}

	return variance / float64(totalCount)
}

// splitBucketWithQualityAndAlpha splits a bucket considering alpha channel.
func splitBucketWithQualityAndAlpha(colors []ColorWithCount, quality float64) ([]ColorWithCount, []ColorWithCount) {
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

	splitRatio := 0.5
	if quality < 0.5 {
		splitRatio = 0.5 - (0.5-quality)*0.3
	} else if quality > 0.5 {
		splitRatio = 0.5 + (quality-0.5)*0.3
	}

	mid := int(float64(len(sorted)) * splitRatio)
	if mid < 1 {
		mid = 1
	}
	if mid >= len(sorted) {
		mid = len(sorted) - 1
	}

	return sorted[:mid], sorted[mid:]
}

// averageColorsWithAlpha calculates the average color including alpha channel.
func averageColorsWithAlpha(colors []ColorWithCount) Color {
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

// ExtendedColor represents an RGBA color.
type ExtendedColor struct {
	R, G, B, A uint8
}

// ExtendedColorWithCount extends ExtendedColor with frequency information.
type ExtendedColorWithCount struct {
	ExtendedColor
	Count int
}

// MedianCutRGBA performs median cut on RGBA images with full alpha support.
// This is more accurate for images with transparency.
func MedianCutRGBA(colorsWithCount []ExtendedColorWithCount, maxColors int) []ExtendedColor {
	return MedianCutRGBAWithQuality(colorsWithCount, maxColors, 1.0)
}

// MedianCutRGBAWithQuality performs median cut on RGBA images with quality control.
func MedianCutRGBAWithQuality(colorsWithCount []ExtendedColorWithCount, maxColors int, quality float64) []ExtendedColor {
	if len(colorsWithCount) == 0 {
		return []ExtendedColor{}
	}

	if len(colorsWithCount) <= maxColors {
		result := make([]ExtendedColor, len(colorsWithCount))
		for i, cwc := range colorsWithCount {
			result[i] = cwc.ExtendedColor
		}
		return result
	}

	effectiveMaxColors := maxColors
	if quality < 1.0 {
		effectiveMaxColors = int(float64(maxColors) * (0.5 + quality*0.5))
		if effectiveMaxColors < 2 {
			effectiveMaxColors = 2
		}
	}

	type rgbaBucket struct {
		colors []ExtendedColorWithCount
	}

	buckets := []rgbaBucket{{colors: colorsWithCount}}

	for len(buckets) < effectiveMaxColors {
		largestIdx := -1
		maxVariance := 0.0
		for i := range buckets {
			variance := calculateRGBVariance(buckets[i].colors)
			if variance > maxVariance && len(buckets[i].colors) >= 2 {
				maxVariance = variance
				largestIdx = i
			}
		}

		if largestIdx == -1 || len(buckets[largestIdx].colors) < 2 {
			break
		}

		left, right := splitRGBBucket(buckets[largestIdx].colors, quality)

		buckets[largestIdx].colors = left
		if len(right) > 0 {
			buckets = append(buckets, rgbaBucket{colors: right})
		}
	}

	result := make([]ExtendedColor, 0, maxColors)
	for _, b := range buckets {
		if len(b.colors) > 0 {
			result = append(result, averageRGBColors(b.colors))
		}
	}

	return result
}

// calculateRGBVariance calculates the color variance in RGB space.
func calculateRGBVariance(colors []ExtendedColorWithCount) float64 {
	if len(colors) == 0 {
		return 0
	}

	var totalR, totalG, totalB int
	totalCount := 0
	for _, c := range colors {
		totalR += int(c.R) * c.Count
		totalG += int(c.G) * c.Count
		totalB += int(c.B) * c.Count
		totalCount += c.Count
	}

	meanR := float64(totalR) / float64(totalCount)
	meanG := float64(totalG) / float64(totalCount)
	meanB := float64(totalB) / float64(totalCount)

	var variance float64
	for _, c := range colors {
		weight := float64(c.Count)
		dr := float64(c.R) - meanR
		dg := float64(c.G) - meanG
		db := float64(c.B) - meanB
		variance += weight * (dr*dr + dg*dg + db*db)
	}

	return variance / float64(totalCount)
}

// splitRGBBucket splits an RGBA bucket into two at the median.
func splitRGBBucket(colors []ExtendedColorWithCount, quality float64) ([]ExtendedColorWithCount, []ExtendedColorWithCount) {
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
		sortBy = 2
	}

	sorted := make([]ExtendedColorWithCount, len(colors))
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

	splitRatio := 0.5
	if quality < 0.5 {
		splitRatio = 0.5 - (0.5-quality)*0.3
	} else if quality > 0.5 {
		splitRatio = 0.5 + (quality-0.5)*0.3
	}

	mid := int(float64(len(sorted)) * splitRatio)
	if mid < 1 {
		mid = 1
	}
	if mid >= len(sorted) {
		mid = len(sorted) - 1
	}

	return sorted[:mid], sorted[mid:]
}

// averageRGBColors calculates the average color of an RGBA bucket.
func averageRGBColors(colors []ExtendedColorWithCount) ExtendedColor {
	var totalR, totalG, totalB int
	var totalCount int

	for _, c := range colors {
		totalR += int(c.R) * c.Count
		totalG += int(c.G) * c.Count
		totalB += int(c.B) * c.Count
		totalCount += c.Count
	}

	if totalCount == 0 {
		totalCount = len(colors)
	}

	return ExtendedColor{
		R: uint8(totalR / totalCount),
		G: uint8(totalG / totalCount),
		B: uint8(totalB / totalCount),
	}
}

// QuantizationError represents the error introduced by quantization.
type QuantizationError struct {
	MaxError float64
	AvgError float64
	RMSE     float64
}

// CalculateQuantizationError calculates error metrics for a quantization result.
func CalculateQuantizationError(original []byte, quantized []byte, palette Palette, colorType int) QuantizationError {
	if len(original) == 0 || len(quantized) == 0 {
		return QuantizationError{
			MaxError: 0,
			AvgError: 0,
			RMSE:     0,
		}
	}

	bpp := BytesPerPixel(ColorType(colorType))
	width := len(original) / bpp

	var maxErr float64
	var totalErr float64
	var sqErr float64

	for i := 0; i < width; i++ {
		offset := i * bpp
		origColor := Color{
			R: original[offset],
			G: original[offset+1],
			B: original[offset+2],
		}

		paletteIdx := int(quantized[i])
		quantColor := palette.Colors[paletteIdx]

		dr := float64(origColor.R) - float64(quantColor.R)
		dg := float64(origColor.G) - float64(quantColor.G)
		db := float64(origColor.B) - float64(quantColor.B)

		err := math.Sqrt(dr*dr + dg*dg + db*db)
		totalErr += err
		sqErr += err * err
		if err > maxErr {
			maxErr = err
		}
	}

	avgErr := totalErr / float64(width)
	rmse := math.Sqrt(sqErr / float64(width))

	return QuantizationError{
		MaxError: maxErr,
		AvgError: avgErr,
		RMSE:     rmse,
	}
}
