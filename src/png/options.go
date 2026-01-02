package png

type Preset int

const (
	PresetFast Preset = iota
	PresetBalanced
	PresetMax
)

type FilterStrategy int

const (
	FilterStrategyNone FilterStrategy = iota
	FilterStrategySub
	FilterStrategyUp
	FilterStrategyAverage
	FilterStrategyPaeth
	FilterStrategyMinSum
	FilterStrategyAdaptive
	FilterStrategyAdaptiveFast
)

type Options struct {
	Width           int
	Height          int
	ColorType       ColorType
	CompressionLevel int
	FilterStrategy  FilterStrategy
	OptimizeAlpha   bool
	ReduceColorType bool
	StripMetadata   bool
	OptimalDeflate  bool
}

func FastOptions(width, height int) Options {
	return Options{
		Width:            width,
		Height:           height,
		ColorType:        ColorRGBA,
		CompressionLevel: 2,
		FilterStrategy:   FilterStrategyMinSum,
		OptimizeAlpha:    false,
		ReduceColorType:  false,
		StripMetadata:    false,
		OptimalDeflate:   false,
	}
}

func BalancedOptions(width, height int) Options {
	return Options{
		Width:            width,
		Height:           height,
		ColorType:        ColorRGBA,
		CompressionLevel: 6,
		FilterStrategy:   FilterStrategyAdaptive,
		OptimizeAlpha:    true,
		ReduceColorType:  true,
		StripMetadata:    true,
		OptimalDeflate:   false,
	}
}

func MaxOptions(width, height int) Options {
	return Options{
		Width:            width,
		Height:           height,
		ColorType:        ColorRGBA,
		CompressionLevel: 9,
		FilterStrategy:   FilterStrategyMinSum,
		OptimizeAlpha:    true,
		ReduceColorType:  true,
		StripMetadata:    true,
		OptimalDeflate:   true,
	}
}
