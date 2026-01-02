package png

type OptionsBuilder struct {
	opts Options
}

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

func (b *OptionsBuilder) Fast() *OptionsBuilder {
	b.opts.CompressionLevel = 2
	b.opts.FilterStrategy = FilterStrategyAdaptiveFast
	b.opts.OptimizeAlpha = false
	b.opts.ReduceColorType = false
	b.opts.StripMetadata = false
	b.opts.OptimalDeflate = false
	return b
}

func (b *OptionsBuilder) Balanced() *OptionsBuilder {
	b.opts.CompressionLevel = 6
	b.opts.FilterStrategy = FilterStrategyAdaptive
	b.opts.OptimizeAlpha = true
	b.opts.ReduceColorType = true
	b.opts.StripMetadata = true
	b.opts.OptimalDeflate = false
	return b
}

func (b *OptionsBuilder) Max() *OptionsBuilder {
	b.opts.CompressionLevel = 9
	b.opts.FilterStrategy = FilterStrategyMinSum
	b.opts.OptimizeAlpha = true
	b.opts.ReduceColorType = true
	b.opts.StripMetadata = true
	b.opts.OptimalDeflate = true
	return b
}

func (b *OptionsBuilder) CompressionLevel(level int) *OptionsBuilder {
	if level < 1 {
		level = 1
	} else if level > 9 {
		level = 9
	}
	b.opts.CompressionLevel = level
	return b
}

func (b *OptionsBuilder) FilterStrategy(strategy FilterStrategy) *OptionsBuilder {
	b.opts.FilterStrategy = strategy
	return b
}

func (b *OptionsBuilder) OptimizeAlpha(enabled bool) *OptionsBuilder {
	b.opts.OptimizeAlpha = enabled
	return b
}

func (b *OptionsBuilder) ReduceColorType(enabled bool) *OptionsBuilder {
	b.opts.ReduceColorType = enabled
	return b
}

func (b *OptionsBuilder) StripMetadata(enabled bool) *OptionsBuilder {
	b.opts.StripMetadata = enabled
	return b
}

func (b *OptionsBuilder) OptimalDeflate(enabled bool) *OptionsBuilder {
	b.opts.OptimalDeflate = enabled
	return b
}

func (b *OptionsBuilder) Build() Options {
	return b.opts
}
