package png

import "github.com/mac/go-pixo/src/compress"

type FilterSelectionConfig struct {
	EarlyTerminationEnabled   bool
	EarlyTerminationThreshold float64
	MinFiltersToTry           int
	FilterSelectionThreshold  float64
}

const (
	DefaultMinSumThreshold  = 256.0
	DefaultEntropyThreshold = 0.5
	DefaultMinFiltersToTry  = 2
)

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
	FilterStrategyBigrams
	FilterStrategyParallel
)

type DistanceMetricType string

const (
	DistanceMetricEuclidean DistanceMetricType = "euclidean"
	DistanceMetricRedmean   DistanceMetricType = "redmean"
)

// ZopfliIterations controls the number of iterations for Zopfli compression.
// More iterations generally yield better compression but increase processing time.
// The tradeoff is approximately logarithmic: 5 iterations provides good results,
// 10 iterations offers a good balance, and 15+ iterations provides marginal gains
// for significantly increased computation time. Each iteration may add 2-5 seconds
// depending on image size. Memory usage scales linearly with iterations.
// Recommended values: 5 for speed-focused, 10 for balanced, 15+ for maximum compression.
//
// ZopfliEnabled enables Zopfli DEFLATE compression algorithm for PNG encoding.
// Zopfli produces smaller DEFLATE streams than standard zlib by performing
// extensive searches for optimal encodings. It is significantly slower than
// standard compression but can reduce file size by 5-15%.
// When disabled (default), standard zlib compression is used.
// Recommended for: archival storage, assets where size matters over encoding time.
//
// ZopfliBlockSplitting controls whether Zopfli uses block splitting optimization.
// Block splitting divides the data into smaller blocks, each with its own optimal
// encoding method. This typically improves compression by 1-5% for typical images.
// Disable this only if you have specific compatibility requirements or are
// experiencing issues with certain decoders. Default is enabled.
//
// ZopfliSplitThreshold sets the threshold for block splitting decisions.
// Values closer to 0 enable more aggressive splitting, while higher values
// (up to 1.0) reduce splitting. The default of 0.001 provides good compression
// without excessive fragmentation. This parameter has minimal effect when
// ZopfliBlockSplitting is disabled.
// Memory and time requirements increase linearly with the number of blocks.
type Options struct {
	Width                    int
	Height                   int
	ColorType                ColorType
	CompressionLevel         int
	FilterStrategy           FilterStrategy
	FilterEarlyTermination   bool
	FilterSelectionThreshold float64
	FilterSelectionConfig    FilterSelectionConfig
	OptimizeAlpha            bool
	ReduceColorType          bool
	StripMetadata            bool
	PreserveChunks           []Chunk
	OptimalDeflate           bool
	OptimalConfig            compress.OptimalConfig
	MaxColors                int
	Dithering                bool
	DitheringStrength        float64
	QualityTarget            int
	ZopfliIterations         int
	ZopfliEnabled            bool
	ZopfliBlockSplitting     bool
	ZopfliSplitThreshold     float64
	EnsureSizeNotLarger      bool
	OriginalFileSize         int
	ProgressCallback         func(phase string, progress int)
	UsePerceptualDistance    bool
	DistanceMetric           DistanceMetricType
}

func FastOptions(width, height int) Options {
	return Options{
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         2,
		FilterStrategy:           FilterStrategyMinSum,
		FilterEarlyTermination:   true,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   true,
			EarlyTerminationThreshold: 0.1,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultMinSumThreshold,
		},
		OptimizeAlpha:         false,
		ReduceColorType:       false,
		StripMetadata:         false,
		OptimalDeflate:        false,
		OptimalConfig:         compress.OptimalConfigForLevel(2),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      0,
		ZopfliEnabled:         false,
		ZopfliBlockSplitting:  false,
		ZopfliSplitThreshold:  0.001,
		EnsureSizeNotLarger:   false,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
}

// FasterOptions returns options optimized for speed with size guarantee.
// If compressed output is larger than original, it will use minimal compression.
func FasterOptions(width, height int) Options {
	return Options{
		Width:                 width,
		Height:                height,
		ColorType:             ColorRGBA,
		CompressionLevel:      1,
		FilterStrategy:        FilterStrategyNone,
		OptimizeAlpha:         false,
		ReduceColorType:       false,
		StripMetadata:         false,
		OptimalDeflate:        false,
		OptimalConfig:         compress.OptimalConfigForLevel(1),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      0,
		ZopfliEnabled:         false,
		ZopfliBlockSplitting:  false,
		ZopfliSplitThreshold:  0.001,
		EnsureSizeNotLarger:   true,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
}

// SmallerOptions returns options optimized for maximum compression with quality preservation.
// Uses entropy-based filtering and moderate Zopfli iterations for best size/quality ratio.
func SmallerOptions(width, height int) Options {
	return Options{
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         9,
		FilterStrategy:           FilterStrategyEntropy,
		FilterEarlyTermination:   false,
		FilterSelectionThreshold: DefaultEntropyThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   false,
			EarlyTerminationThreshold: 0.05,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultEntropyThreshold,
		},
		OptimizeAlpha:         true,
		ReduceColorType:       true,
		StripMetadata:         true,
		OptimalDeflate:        true,
		OptimalConfig:         compress.OptimalConfigForLevel(9),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      10,
		ZopfliEnabled:         true,
		ZopfliBlockSplitting:  true,
		ZopfliSplitThreshold:  0.001,
		EnsureSizeNotLarger:   false,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
}

func BalancedOptions(width, height int) Options {
	return Options{
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         6,
		FilterStrategy:           FilterStrategyAdaptive,
		FilterEarlyTermination:   true,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   true,
			EarlyTerminationThreshold: 0.1,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultMinSumThreshold,
		},
		OptimizeAlpha:         true,
		ReduceColorType:       true,
		StripMetadata:         true,
		OptimalDeflate:        false,
		OptimalConfig:         compress.OptimalConfigForLevel(6),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      0,
		ZopfliEnabled:         false,
		ZopfliBlockSplitting:  true,
		ZopfliSplitThreshold:  0.001,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
}

func MaxOptions(width, height int) Options {
	return Options{
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         9,
		FilterStrategy:           FilterStrategyMinSum,
		FilterEarlyTermination:   false,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   false,
			EarlyTerminationThreshold: 0.05,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultMinSumThreshold,
		},
		OptimizeAlpha:         true,
		ReduceColorType:       true,
		StripMetadata:         true,
		OptimalDeflate:        true,
		OptimalConfig:         compress.OptimalConfigForLevel(9),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      12,
		ZopfliEnabled:         true,
		ZopfliBlockSplitting:  true,
		ZopfliSplitThreshold:  0.001,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
	}
}

func ExtremeOptions(width, height int) Options {
	return Options{
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         10,
		FilterStrategy:           FilterStrategyEntropy,
		FilterEarlyTermination:   false,
		FilterSelectionThreshold: DefaultEntropyThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   false,
			EarlyTerminationThreshold: 0.01,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultEntropyThreshold,
		},
		OptimizeAlpha:         true,
		ReduceColorType:       true,
		StripMetadata:         true,
		OptimalDeflate:        true,
		OptimalConfig:         compress.OptimalConfigForLevel(9),
		MaxColors:             0,
		Dithering:             false,
		DitheringStrength:     0.0,
		QualityTarget:         100,
		ZopfliIterations:      15,
		ZopfliEnabled:         true,
		ZopfliBlockSplitting:  true,
		ZopfliSplitThreshold:  0.001,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
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
		Width:                    width,
		Height:                   height,
		ColorType:                ColorRGBA,
		CompressionLevel:         9,
		FilterStrategy:           FilterStrategyMinSum,
		FilterEarlyTermination:   true,
		FilterSelectionThreshold: DefaultMinSumThreshold,
		FilterSelectionConfig: FilterSelectionConfig{
			EarlyTerminationEnabled:   true,
			EarlyTerminationThreshold: 0.1,
			MinFiltersToTry:           DefaultMinFiltersToTry,
			FilterSelectionThreshold:  DefaultMinSumThreshold,
		},
		OptimizeAlpha:         true,
		ReduceColorType:       false,
		StripMetadata:         true,
		OptimalDeflate:        true,
		OptimalConfig:         compress.OptimalConfigForLevel(9),
		MaxColors:             maxColors,
		Dithering:             true,
		DitheringStrength:     0.5,
		QualityTarget:         75,
		ZopfliIterations:      0,
		ZopfliEnabled:         false,
		ZopfliBlockSplitting:  true,
		ZopfliSplitThreshold:  0.001,
		UsePerceptualDistance: false,
		DistanceMetric:        DistanceMetricEuclidean,
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

// WithZopfliIterations sets the number of Zopfli iterations for compression.
// More iterations yield better compression but increase processing time.
// The tradeoff is approximately logarithmic: 5 iterations provides good results,
// 10 iterations offers a balance, and 15+ iterations provides marginal gains.
// Each iteration adds 2-5 seconds depending on image size.
func (o *Options) WithZopfliIterations(n int) *Options {
	o.ZopfliIterations = n
	return o
}

// WithZopfliEnabled enables or disables Zopfli DEFLATE compression.
// Zopfli produces 5-15% smaller files but is significantly slower.
// Default is false (standard zlib compression).
func (o *Options) WithZopfliEnabled(b bool) *Options {
	o.ZopfliEnabled = b
	return o
}

// WithZopfliBlockSplitting enables or disables Zopfli block splitting optimization.
// Block splitting typically improves compression by 1-5% for typical images.
// Default is true.
func (o *Options) WithZopfliBlockSplitting(b bool) *Options {
	o.ZopfliBlockSplitting = b
	return o
}
