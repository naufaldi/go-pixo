package png

import "math"

func ColorWeight(c Color) float64 {
	rMean := float64(c.R) / 2.0
	return 1.0 + rMean/256.0
}

func RedmeanDistanceSq(c1, c2 Color) float64 {
	dr := float64(int64(c1.R) - int64(c2.R))
	dg := float64(int64(c1.G) - int64(c2.G))
	db := float64(int64(c1.B) - int64(c2.B))

	rMean := (float64(c1.R) + float64(c2.R)) / 2.0
	weightR := 1.0 + rMean/256.0

	return 2.0*weightR*dr*dr + 4.0*dg*dg + (3.0+weightR)*db*db
}

func RedmeanDistanceSqUint64(c1, c2 Color) uint64 {
	dr := float64(int64(c1.R) - int64(c2.R))
	dg := float64(int64(c1.G) - int64(c2.G))
	db := float64(int64(c1.B) - int64(c2.B))

	rMean := (float64(c1.R) + float64(c2.R)) / 2.0
	weightR := 1.0 + rMean/256.0

	return uint64(2.0*weightR*dr*dr + 4.0*dg*dg + (3.0+weightR)*db*db)
}

func RedmeanDistance(c1, c2 Color) float64 {
	return math.Sqrt(RedmeanDistanceSq(c1, c2))
}

func WeightedColorDistance(c1, c2 Color) float64 {
	return RedmeanDistance(c1, c2)
}
