package png

type Preset int

const (
	PresetFast Preset = iota
	PresetBalanced
	PresetMax
	PresetExtreme
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
	FilterStrategyEntropy
	FilterStrategyBruteForce
)

// Options represents PNG encoding options.
type Options struct {
	Width             int
	Height            int
	ColorType         ColorType
	CompressionLevel  int
	FilterStrategy    FilterStrategy
	OptimizeAlpha     bool
	ReduceColorType   bool
	StripMetadata     bool
	OptimalDeflate    bool
	MaxColors         int
	Dithering         bool
	DitheringStrength float64
	QualityTarget     int
	ZopfliIterations  int
}

func FastOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  2,
		FilterStrategy:    FilterStrategyMinSum,
		OptimizeAlpha:     false,
		ReduceColorType:   false,
		StripMetadata:     false,
		OptimalDeflate:    false,
		MaxColors:         0,
		Dithering:         false,
		DitheringStrength: 0.0,
		QualityTarget:     100,
		ZopfliIterations:  0,
	}
}

func BalancedOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  6,
		FilterStrategy:    FilterStrategyAdaptive,
		OptimizeAlpha:     true,
		ReduceColorType:   true,
		StripMetadata:     true,
		OptimalDeflate:    false,
		MaxColors:         0,
		Dithering:         false,
		DitheringStrength: 0.0,
		QualityTarget:     100,
		ZopfliIterations:  0,
	}
}

func MaxOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  9,
		FilterStrategy:    FilterStrategyMinSum,
		OptimizeAlpha:     true,
		ReduceColorType:   true,
		StripMetadata:     true,
		OptimalDeflate:    true,
		MaxColors:         0,
		Dithering:         false,
		DitheringStrength: 0.0,
		QualityTarget:     100,
		ZopfliIterations:  5,
	}
}

func ExtremeOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  10,
		FilterStrategy:    FilterStrategyEntropy,
		OptimizeAlpha:     true,
		ReduceColorType:   true,
		StripMetadata:     true,
		OptimalDeflate:    true,
		MaxColors:         0,
		Dithering:         false,
		DitheringStrength: 0.0,
		QualityTarget:     100,
		ZopfliIterations:  15,
	}
}

func LossyOptions(width, height int, maxColors int) Options {
	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  9,
		FilterStrategy:    FilterStrategyMinSum,
		OptimizeAlpha:     true,
		ReduceColorType:   false,
		StripMetadata:     true,
		OptimalDeflate:    true,
		MaxColors:         maxColors,
		Dithering:         true,
		DitheringStrength: 0.5,
		QualityTarget:     75,
		ZopfliIterations:  0,
	}
}

// QualityPresets for lossy compression with different quality targets.
// Note: This function requires width and height to be passed.
// Use lossyWithQuality(width, height, maxColors, quality) directly.
func QualityPresets() map[string]Options {
	// Return a map of preset names to configuration functions
	// This is a placeholder - actual usage requires width/height
	return map[string]Options{}
}

func lossyWithQuality(width, height, maxColors, quality int) Options {
	opts := LossyOptions(width, height, maxColors)
	opts.QualityTarget = quality
	if quality >= 90 {
		opts.DitheringStrength = 0.25
	} else if quality >= 75 {
		opts.DitheringStrength = 0.5
	} else if quality >= 50 {
		opts.DitheringStrength = 0.75
	} else {
		opts.DitheringStrength = 1.0
	}
	return opts
}

// ApplyLossy applies lossy compression settings to an Options struct.
func (o *Options) ApplyLossy(maxColors int, qualityTarget int, ditheringStrength float64) {
	if maxColors <= 0 {
		maxColors = 256
	}
	if maxColors > 256 {
		maxColors = 256
	}
	if qualityTarget < 0 {
		qualityTarget = 0
	}
	if qualityTarget > 100 {
		qualityTarget = 100
	}
	if ditheringStrength < 0 {
		ditheringStrength = 0
	}
	if ditheringStrength > 1 {
		ditheringStrength = 1
	}

	o.MaxColors = maxColors
	o.Dithering = true
	o.DitheringStrength = ditheringStrength
	o.QualityTarget = qualityTarget
}

// ApplyLossless resets lossy settings to lossless mode.
func (o *Options) ApplyLossless() {
	o.MaxColors = 0
	o.Dithering = false
	o.DitheringStrength = 0.0
	o.QualityTarget = 100
}

// CompressionRatio returns the estimated compression ratio based on options.
func (o *Options) CompressionRatio() float64 {
	if o.ZopfliIterations >= 15 {
		return 0.45
	}
	if o.ZopfliIterations >= 5 {
		return 0.5
	}
	if o.OptimalDeflate {
		return 0.55
	}
	if o.CompressionLevel >= 6 {
		return 0.6
	}
	return 0.7
}

// IsLossy returns true if the options enable lossy compression.
func (o *Options) IsLossy() bool {
	return o.MaxColors > 0 && o.MaxColors < 256
}
