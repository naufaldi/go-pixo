package png

import "github.com/mac/go-pixo/src/compress"

// Preset selects a predefined set of encoding options.
type Preset int

const (
	// PresetFast favors speed over size.
	PresetFast Preset = iota
	// PresetBalanced balances speed and size.
	PresetBalanced
	// PresetMax favors size over speed.
	PresetMax
	// PresetExtreme favors maximum size reduction.
	PresetExtreme
	// PresetUltra uses combined Entropy+Bigrams filter strategy with deep Zopfli passes.
	PresetUltra
)

// FilterStrategy selects the scanline filter strategy.
type FilterStrategy int

const (
	// FilterStrategyNone selects the None filter.
	FilterStrategyNone FilterStrategy = iota
	// FilterStrategySub selects the Sub filter.
	FilterStrategySub
	// FilterStrategyUp selects the Up filter.
	FilterStrategyUp
	// FilterStrategyAverage selects the Average filter.
	FilterStrategyAverage
	// FilterStrategyPaeth selects the Paeth filter.
	FilterStrategyPaeth
	// FilterStrategyMinSum selects the filter with minimum sum of absolute values.
	FilterStrategyMinSum
	// FilterStrategyAdaptive is the default adaptive strategy.
	FilterStrategyAdaptive
	// FilterStrategyAdaptiveFast is an adaptive strategy optimized for speed.
	FilterStrategyAdaptiveFast
	// FilterStrategyEntropy selects the filter with minimum entropy.
	FilterStrategyEntropy
	// FilterStrategyBruteForce tries all filters for each row.
	FilterStrategyBruteForce
	// FilterStrategyBigrams selects the filter with minimum distinct bigrams.
	FilterStrategyBigrams
	// FilterStrategyParallel selects filters in parallel (adaptive strategy).
	FilterStrategyParallel
	// FilterStrategyCombined scores each filter with a combined Entropy + Bigrams rank.
	FilterStrategyCombined
)

// DistanceMetric selects the color distance formula for quantization.
type DistanceMetric int

const (
	DistanceMetricEuclidean DistanceMetric = iota
	DistanceMetricRedmean
)

// Options represents PNG encoding options.
type Options struct {
	Width               int
	Height              int
	ColorType           ColorType
	CompressionLevel    int
	FilterStrategy      FilterStrategy
	OptimizeAlpha       bool
	ReduceColorType     bool
	StripMetadata       bool
	OptimalDeflate      bool
	MaxColors           int
	Dithering           bool
	DitheringStrength   float64
	QualityTarget       int
	ZopfliIterations    int
	EnsureSizeNotLarger bool                             // If true, ensure output is not larger than original
	PreserveChunks      []Chunk
	OriginalFileSize    int
	UsePerceptualDistance bool
	DistanceMetric        DistanceMetric
	OptimalConfig         compress.OptimalConfig
	ProgressCallback    func(phase string, progress int) // Optional callback for progress updates
}

// FastOptions returns options optimized for speed.
func FastOptions(width, height int) Options {
	return Options{
		Width:               width,
		Height:              height,
		ColorType:           ColorRGBA,
		CompressionLevel:    2,
		FilterStrategy:      FilterStrategyMinSum,
		OptimizeAlpha:       false,
		ReduceColorType:     false,
		StripMetadata:       false,
		OptimalDeflate:      false,
		MaxColors:           0,
		Dithering:           false,
		DitheringStrength:   0.0,
		QualityTarget:       100,
		ZopfliIterations:    0,
		EnsureSizeNotLarger: false,
	}
}

// FasterOptions returns options optimized for speed with size guarantee.
// If compressed output is larger than original, it will use minimal compression.
func FasterOptions(width, height int) Options {
	return Options{
		Width:               width,
		Height:              height,
		ColorType:           ColorRGBA,
		CompressionLevel:    1,
		FilterStrategy:      FilterStrategyNone,
		OptimizeAlpha:       false,
		ReduceColorType:     false,
		StripMetadata:       false,
		OptimalDeflate:      false,
		MaxColors:           0,
		Dithering:           false,
		DitheringStrength:   0.0,
		QualityTarget:       100,
		ZopfliIterations:    0,
		EnsureSizeNotLarger: true,
	}
}

// SmallerOptions returns options optimized for maximum compression with quality preservation.
// Uses bigrams-based filtering and moderate Zopfli iterations for best size/quality ratio.
func SmallerOptions(width, height int) Options {
	return Options{
		Width:               width,
		Height:              height,
		ColorType:           ColorRGBA,
		CompressionLevel:    9,
		FilterStrategy:      FilterStrategyBigrams,
		OptimizeAlpha:       true,
		ReduceColorType:     true,
		StripMetadata:       true,
		OptimalDeflate:      true,
		MaxColors:           0,
		Dithering:           false,
		DitheringStrength:   0.0,
		QualityTarget:       100,
		ZopfliIterations:    10,
		EnsureSizeNotLarger: false,
	}
}

// BalancedOptions returns a reasonable default Options preset.
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

// MaxOptions returns options tuned for maximum compression.
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

// ExtremeOptions returns options tuned for extreme compression.
func ExtremeOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  10,
		FilterStrategy:    FilterStrategyBigrams,
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

// UltraOptions returns options for maximum compression using the combined Entropy+Bigrams
// filter strategy with deep Zopfli passes. Slower than ExtremeOptions but can squeeze
// out additional savings on images already processed by other tools.
func UltraOptions(width, height int) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGBA,
		CompressionLevel:  10,
		FilterStrategy:    FilterStrategyCombined,
		OptimizeAlpha:     true,
		ReduceColorType:   true,
		StripMetadata:     true,
		OptimalDeflate:    true,
		MaxColors:         0,
		Dithering:         false,
		DitheringStrength: 0.0,
		QualityTarget:     100,
		ZopfliIterations:  20,
		EnsureSizeNotLarger: false,
	}
}

// LossyOptions returns options configured for palette-based lossy compression.
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
