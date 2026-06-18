package jpeg

// Preset represents a JPEG compression preset.
type Preset int

const (
	PresetFast Preset = iota
	PresetBalanced
	PresetMax
)

// QualityPreset represents the quality level for JPEG encoding.
type QualityPreset int

const (
	QualityLow     QualityPreset = iota // Quality ~60, smaller files
	QualityMedium                       // Quality ~75, balanced
	QualityHigh                         // Quality ~85, better quality
	QualityMaximum                      // Quality ~95, best quality
)

// Options represents JPEG encoding options.
type Options struct {
	Width             int
	Height            int
	ColorType         ColorType
	Quality           uint8
	Subsampling       Subsampling
	OptimizeHuffman   bool
	Progressive       bool
	TrellisQuant      bool
	TrellisLambda     float64
	TrellisPerceptual bool
	RestartInterval   *uint16
	UseSIMD           bool
}

// FastOptions returns options optimized for speed.
// Uses 4:4:4 subsampling, standard tables, and baseline encoding.
func FastOptions(width, height int, quality uint8) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGB,
		Quality:           quality,
		Subsampling:       Subsampling444,
		OptimizeHuffman:   false,
		Progressive:       false,
		TrellisQuant:      false,
		TrellisLambda:     1.0,
		TrellisPerceptual: true,
		RestartInterval:   nil,
		UseSIMD:           true,
	}
}

// BalancedOptions returns balanced encoding options.
// Uses 4:2:0 subsampling, standard tables, and baseline encoding.
func BalancedOptions(width, height int, quality uint8) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGB,
		Quality:           quality,
		Subsampling:       Subsampling420,
		OptimizeHuffman:   false,
		Progressive:       false,
		TrellisQuant:      false,
		TrellisLambda:     1.0,
		TrellisPerceptual: true,
		RestartInterval:   nil,
		UseSIMD:           true,
	}
}

// MaxOptions returns options for maximum compression.
// Uses 4:2:0 subsampling, progressive encoding, and trellis quantization.
// Note: Optimized Huffman is disabled when trellis is enabled because the
// optimized tables are built based on standard quantization, but trellis
// produces different coefficients that would mismatch.
func MaxOptions(width, height int, quality uint8) Options {
	return Options{
		Width:             width,
		Height:            height,
		ColorType:         ColorRGB,
		Quality:           quality,
		Subsampling:       Subsampling420,
		OptimizeHuffman:   true,
		Progressive:       true,
		TrellisQuant:      true,
		TrellisLambda:     0,
		TrellisPerceptual: true,
		RestartInterval:   nil,
		UseSIMD:           true,
	}
}

// GetQualityFromPreset returns the quality value for a given preset.
func GetQualityFromPreset(preset QualityPreset) uint8 {
	switch preset {
	case QualityLow:
		return 60
	case QualityMedium:
		return 75
	case QualityHigh:
		return 85
	case QualityMaximum:
		return 95
	default:
		return 75
	}
}
