package jpeg

// OptionsBuilder provides a fluent API for building JPEG options.
type OptionsBuilder struct {
	opts Options
}

// NewOptionsBuilder creates a new OptionsBuilder with default balanced options.
func NewOptionsBuilder(width, height int) *OptionsBuilder {
	return &OptionsBuilder{
		opts: BalancedOptions(width, height, 75),
	}
}

// Quality sets the JPEG quality (1-100).
func (b *OptionsBuilder) Quality(quality uint8) *OptionsBuilder {
	if quality < 1 {
		quality = 1
	} else if quality > 100 {
		quality = 100
	}
	b.opts.Quality = quality
	return b
}

// Subsampling sets the chroma subsampling mode.
func (b *OptionsBuilder) Subsampling(subsampling Subsampling) *OptionsBuilder {
	b.opts.Subsampling = subsampling
	return b
}

// OptimizeHuffman enables or disables Huffman table optimization.
func (b *OptionsBuilder) OptimizeHuffman(optimize bool) *OptionsBuilder {
	b.opts.OptimizeHuffman = optimize
	return b
}

// Progressive enables or disables progressive encoding.
func (b *OptionsBuilder) Progressive(progressive bool) *OptionsBuilder {
	b.opts.Progressive = progressive
	return b
}

// TrellisQuant enables or disables trellis quantization.
func (b *OptionsBuilder) TrellisQuant(trellis bool) *OptionsBuilder {
	b.opts.TrellisQuant = trellis
	return b
}

// RestartInterval sets the restart interval in MCUs.
func (b *OptionsBuilder) RestartInterval(interval uint16) *OptionsBuilder {
	b.opts.RestartInterval = &interval
	return b
}

// ColorType sets the color type.
func (b *OptionsBuilder) ColorType(colorType ColorType) *OptionsBuilder {
	b.opts.ColorType = colorType
	return b
}

// Preset applies a preset configuration.
func (b *OptionsBuilder) Preset(preset Preset) *OptionsBuilder {
	switch preset {
	case PresetFast:
		b.opts = FastOptions(b.opts.Width, b.opts.Height, b.opts.Quality)
	case PresetBalanced:
		b.opts = BalancedOptions(b.opts.Width, b.opts.Height, b.opts.Quality)
	case PresetMax:
		b.opts = MaxOptions(b.opts.Width, b.opts.Height, b.opts.Quality)
	}
	return b
}

// Build returns the final Options.
func (b *OptionsBuilder) Build() Options {
	return b.opts
}
