package png

// OptionsBuilder provides a fluent API for constructing Options values.
type OptionsBuilder struct {
	opts Options
}

// NewOptionsBuilder creates an OptionsBuilder for an image of the given size.
func NewOptionsBuilder(width, height int) *OptionsBuilder {
	return &OptionsBuilder{
		opts: Options{
			Width:            width,
			Height:           height,
			ColorType:        ColorRGBA,
			CompressionLevel: 6,
			FilterStrategy:   FilterStrategyAdaptive,
			OptimizeAlpha:    true,
			ReduceColorType:  true,
			StripMetadata:    true,
			OptimalDeflate:   false,
		},
	}
}

// Fast configures the builder for fast encoding.
func (b *OptionsBuilder) Fast() *OptionsBuilder {
	b.opts.CompressionLevel = 2
	b.opts.FilterStrategy = FilterStrategyAdaptiveFast
	b.opts.OptimizeAlpha = false
	b.opts.ReduceColorType = false
	b.opts.StripMetadata = false
	b.opts.OptimalDeflate = false
	return b
}

// Balanced configures the builder for balanced speed and size.
func (b *OptionsBuilder) Balanced() *OptionsBuilder {
	b.opts.CompressionLevel = 6
	b.opts.FilterStrategy = FilterStrategyAdaptive
	b.opts.OptimizeAlpha = true
	b.opts.ReduceColorType = true
	b.opts.StripMetadata = true
	b.opts.OptimalDeflate = false
	return b
}

// Max configures the builder for maximum compression.
func (b *OptionsBuilder) Max() *OptionsBuilder {
	b.opts.CompressionLevel = 9
	b.opts.FilterStrategy = FilterStrategyMinSum
	b.opts.OptimizeAlpha = true
	b.opts.ReduceColorType = true
	b.opts.StripMetadata = true
	b.opts.OptimalDeflate = true
	return b
}

// CompressionLevel sets the DEFLATE compression level.
func (b *OptionsBuilder) CompressionLevel(level int) *OptionsBuilder {
	if level < 1 {
		level = 1
	} else if level > 9 {
		level = 9
	}
	b.opts.CompressionLevel = level
	return b
}

// FilterStrategy sets the PNG filter selection strategy.
func (b *OptionsBuilder) FilterStrategy(strategy FilterStrategy) *OptionsBuilder {
	b.opts.FilterStrategy = strategy
	return b
}

// OptimizeAlpha enables or disables alpha-channel optimizations.
func (b *OptionsBuilder) OptimizeAlpha(enabled bool) *OptionsBuilder {
	b.opts.OptimizeAlpha = enabled
	return b
}

// ReduceColorType enables or disables lossless color type reduction.
func (b *OptionsBuilder) ReduceColorType(enabled bool) *OptionsBuilder {
	b.opts.ReduceColorType = enabled
	return b
}

// StripMetadata enables or disables ancillary metadata removal.
func (b *OptionsBuilder) StripMetadata(enabled bool) *OptionsBuilder {
	b.opts.StripMetadata = enabled
	return b
}

// OptimalDeflate enables or disables the slower "optimal" DEFLATE encoder.
func (b *OptionsBuilder) OptimalDeflate(enabled bool) *OptionsBuilder {
	b.opts.OptimalDeflate = enabled
	return b
}

// Build returns the constructed Options.
func (b *OptionsBuilder) Build() Options {
	return b.opts
}
