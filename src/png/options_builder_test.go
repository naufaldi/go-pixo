package png

import (
	"testing"
)

func TestFastOptions(t *testing.T) {
	opts := FastOptions(100, 100)

	if opts.CompressionLevel != 2 {
		t.Errorf("expected compression level 2, got %d", opts.CompressionLevel)
	}
	if opts.FilterStrategy != FilterStrategyMinSum {
		t.Errorf("expected filter strategy MinSum, got %v", opts.FilterStrategy)
	}
	if opts.OptimizeAlpha != false {
		t.Error("expected OptimizeAlpha to be false")
	}
	if opts.ReduceColorType != false {
		t.Error("expected ReduceColorType to be false")
	}
	if opts.StripMetadata != false {
		t.Error("expected StripMetadata to be false")
	}
	if opts.OptimalDeflate != false {
		t.Error("expected OptimalDeflate to be false")
	}
}

func TestBalancedOptions(t *testing.T) {
	opts := BalancedOptions(100, 100)

	if opts.CompressionLevel != 6 {
		t.Errorf("expected compression level 6, got %d", opts.CompressionLevel)
	}
	if opts.FilterStrategy != FilterStrategyAdaptive {
		t.Errorf("expected filter strategy Adaptive, got %v", opts.FilterStrategy)
	}
	if opts.OptimizeAlpha != true {
		t.Error("expected OptimizeAlpha to be true")
	}
	if opts.ReduceColorType != true {
		t.Error("expected ReduceColorType to be true")
	}
	if opts.StripMetadata != true {
		t.Error("expected StripMetadata to be true")
	}
	if opts.OptimalDeflate != false {
		t.Error("expected OptimalDeflate to be false")
	}
}

func TestMaxOptions(t *testing.T) {
	opts := MaxOptions(100, 100)

	if opts.CompressionLevel != 9 {
		t.Errorf("expected compression level 9, got %d", opts.CompressionLevel)
	}
	if opts.FilterStrategy != FilterStrategyMinSum {
		t.Errorf("expected filter strategy MinSum, got %v", opts.FilterStrategy)
	}
	if opts.OptimizeAlpha != true {
		t.Error("expected OptimizeAlpha to be true")
	}
	if opts.ReduceColorType != true {
		t.Error("expected ReduceColorType to be true")
	}
	if opts.StripMetadata != true {
		t.Error("expected StripMetadata to be true")
	}
	if opts.OptimalDeflate != true {
		t.Error("expected OptimalDeflate to be true")
	}
}

func TestOptionsBuilderDefaults(t *testing.T) {
	builder := NewOptionsBuilder(100, 100)
	opts := builder.Build()

	if opts.Width != 100 {
		t.Errorf("expected width 100, got %d", opts.Width)
	}
	if opts.Height != 100 {
		t.Errorf("expected height 100, got %d", opts.Height)
	}
	if opts.ColorType != ColorRGBA {
		t.Errorf("expected color type RGBA, got %v", opts.ColorType)
	}
}

func TestOptionsBuilderChaining(t *testing.T) {
	opts := NewOptionsBuilder(200, 150).
		CompressionLevel(5).
		FilterStrategy(FilterStrategyNone).
		OptimizeAlpha(true).
		ReduceColorType(false).
		Build()

	if opts.Width != 200 {
		t.Errorf("expected width 200, got %d", opts.Width)
	}
	if opts.Height != 150 {
		t.Errorf("expected height 150, got %d", opts.Height)
	}
	if opts.CompressionLevel != 5 {
		t.Errorf("expected compression level 5, got %d", opts.CompressionLevel)
	}
	if opts.FilterStrategy != FilterStrategyNone {
		t.Errorf("expected filter strategy None, got %v", opts.FilterStrategy)
	}
	if opts.OptimizeAlpha != true {
		t.Error("expected OptimizeAlpha to be true")
	}
	if opts.ReduceColorType != false {
		t.Error("expected ReduceColorType to be false")
	}
}

func TestOptionsBuilderCompressionLevelClamping(t *testing.T) {
	t.Run("below minimum", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).
			CompressionLevel(0).
			Build()
		if opts.CompressionLevel != 1 {
			t.Errorf("expected compression level 1, got %d", opts.CompressionLevel)
		}
	})

	t.Run("above maximum", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).
			CompressionLevel(15).
			Build()
		if opts.CompressionLevel != 9 {
			t.Errorf("expected compression level 9, got %d", opts.CompressionLevel)
		}
	})

	t.Run("within range", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).
			CompressionLevel(7).
			Build()
		if opts.CompressionLevel != 7 {
			t.Errorf("expected compression level 7, got %d", opts.CompressionLevel)
		}
	})
}

func TestOptionsBuilderPresetMethods(t *testing.T) {
	t.Run("Fast preset", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).Fast().Build()
		if opts.CompressionLevel != 2 {
			t.Errorf("expected compression level 2, got %d", opts.CompressionLevel)
		}
	})

	t.Run("Balanced preset", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).Balanced().Build()
		if opts.CompressionLevel != 6 {
			t.Errorf("expected compression level 6, got %d", opts.CompressionLevel)
		}
	})

	t.Run("Max preset", func(t *testing.T) {
		opts := NewOptionsBuilder(100, 100).Max().Build()
		if opts.CompressionLevel != 9 {
			t.Errorf("expected compression level 9, got %d", opts.CompressionLevel)
		}
	})
}
