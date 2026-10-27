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
	// PresetUltra uses combined Entropy+Bigrams filtering and the capped parse search.
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
	// FilterStrategyBruteForce compares five independent, BestSpeed DEFLATE row
	// streams. It is opt-in and does not search combinations across rows.
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

// Supported quantization distance metrics: plain RGB Euclidean and red-mean perceptual.
const (
	DistanceMetricEuclidean DistanceMetric = iota
	DistanceMetricRedmean
)

// Options represents PNG encoding options.
type Options struct {
	Width            int
	Height           int
	ColorType        ColorType
	CompressionLevel int // Zero writes stored blocks directly, even with OptimalDeflate.
	FilterStrategy   FilterStrategy
	// FilterEarlyTermination stops MinSum/Adaptive/Parallel or Entropy scoring
	// after at least two candidates when residuals average <=1 or entropy <=0.5
	// bits/byte, respectively. Other strategies reject this option.
	FilterEarlyTermination bool
	OptimizeAlpha          bool
	ReduceColorType        bool
	StripMetadata          bool
	OptimalDeflate         bool
	MaxColors              int
	Dithering              bool
	DitheringStrength      float64 // 0..1 horizontal error diffusion, only when Dithering is enabled.
	// QualityTarget is the median-cut heuristic (0..100), not a fidelity floor.
	// 100 uses the full MaxColors budget but still permits lossy quantization.
	QualityTarget int
	// QualityTargetSet distinguishes explicit quality 0 from legacy zero-value
	// Options (full quality). Presets and ApplyLossy set it automatically.
	// Nonzero QualityTarget values always take effect.
	QualityTargetSet bool
	// ZopfliIterations budgets up to three distinct parses beyond the baseline;
	// see compress.ZopfliConfig. OptimalConfig.MaxIterations takes precedence.
	ZopfliIterations      int
	EnsureSizeNotLarger   bool // Requests a stored fallback when the candidate misses the size target.
	PreserveChunks        []Chunk
	OriginalFileSize      int
	UsePerceptualDistance bool
	DistanceMetric        DistanceMetric
	// OptimalConfig supplies MaxIterations to the bounded search. BlockSplitting
	// is unsupported; other fields configure compress.OptimalParse, not this path.
	OptimalConfig compress.OptimalConfig
	// ProgressCallback reports phase-local completed work (0..100). Histogram and
	// quantize count pixels, palette counts splits/entries, filtering counts rows,
	// and deflate counts completed optimization passes when available. Other phases
	// only report boundaries and must be displayed as indeterminate during work.
	ProgressCallback func(phase string, progress int)
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
		QualityTargetSet:    true,
		ZopfliIterations:    0,
		EnsureSizeNotLarger: false,
	}
}

// FasterOptions returns options optimized for speed with a cheap stored fallback.
// RecompressPNGBytesLossless provides the file-size guarantee using original bytes.
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
		QualityTargetSet:    true,
		ZopfliIterations:    0,
		EnsureSizeNotLarger: true,
	}
}

// SmallerOptions returns options optimized for maximum compression with quality preservation.
// Uses bigrams-based filtering and the bounded parse search.
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
		QualityTargetSet:    true,
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
		QualityTargetSet:  true,
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
		QualityTargetSet:  true,
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
		QualityTargetSet:  true,
		ZopfliIterations:  15,
	}
}

// UltraOptions returns options for maximum compression using the combined Entropy+Bigrams
// filter strategy and the bounded parse search. Its filter scoring can find
// additional savings; iteration budgets above three do not add compression work.
func UltraOptions(width, height int) Options {
	return Options{
		Width:               width,
		Height:              height,
		ColorType:           ColorRGBA,
		CompressionLevel:    10,
		FilterStrategy:      FilterStrategyCombined,
		OptimizeAlpha:       true,
		ReduceColorType:     true,
		StripMetadata:       true,
		OptimalDeflate:      true,
		MaxColors:           0,
		Dithering:           false,
		DitheringStrength:   0.0,
		QualityTarget:       100,
		QualityTargetSet:    true,
		ZopfliIterations:    20,
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
		QualityTargetSet:  true,
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
	o.QualityTargetSet = true
}

// ApplyLossless resets lossy settings to lossless mode.
func (o *Options) ApplyLossless() {
	o.MaxColors = 0
	o.Dithering = false
	o.DitheringStrength = 0.0
	o.QualityTarget = 100
	o.QualityTargetSet = true
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
