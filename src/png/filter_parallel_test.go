package png

import (
	"runtime"
	"sync"
	"testing"
)

func TestSelectAllParallelBasic(t *testing.T) {
	width, height, bpp := 8, 8, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)
	parallelFilters := SelectAllParallel(pixels, width, height, bpp)

	if len(sequentialFilters) != len(parallelFilters) {
		t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
	}

	for i := range sequentialFilters {
		if sequentialFilters[i] != parallelFilters[i] {
			t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
		}
	}
}

func TestSelectAllParallelSmallImage(t *testing.T) {
	width, height, bpp := 4, 16, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)
	parallelFilters := SelectAllParallel(pixels, width, height, bpp)

	if len(sequentialFilters) != len(parallelFilters) {
		t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
	}

	for i := range sequentialFilters {
		if sequentialFilters[i] != parallelFilters[i] {
			t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
		}
	}
}

func TestSelectAllParallelLargeImage(t *testing.T) {
	width, height, bpp := 64, 256, 4
	pixels := make([]byte, width*height*bpp)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := (y*width + x) * bpp
			pixels[offset] = byte(x * 4)
			pixels[offset+1] = byte(y * 4)
			pixels[offset+2] = byte((x + y) * 2)
			pixels[offset+3] = 255
		}
	}

	sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)
	parallelFilters := SelectAllParallel(pixels, width, height, bpp)

	if len(sequentialFilters) != len(parallelFilters) {
		t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
	}

	for i := range sequentialFilters {
		if sequentialFilters[i] != parallelFilters[i] {
			t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
		}
	}
}

func TestSelectAllParallelDifferentBPP(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"grayscale 1bpp", 16, 64, 1},
		{"RGB 3bpp", 16, 64, 3},
		{"RGBA 4bpp", 16, 64, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pixels := make([]byte, tc.width*tc.height*tc.bpp)
			for i := range pixels {
				pixels[i] = byte(i % 256)
			}

			sequentialFilters := SelectAllWithStrategy(pixels, tc.width, tc.height, tc.bpp, FilterStrategyAdaptive)
			parallelFilters := SelectAllParallel(pixels, tc.width, tc.height, tc.bpp)

			if len(sequentialFilters) != len(parallelFilters) {
				t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
			}

			for i := range sequentialFilters {
				if sequentialFilters[i] != parallelFilters[i] {
					t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
				}
			}
		})
	}
}

func strategyName(s FilterStrategy) string {
	switch s {
	case FilterStrategyNone:
		return "None"
	case FilterStrategySub:
		return "Sub"
	case FilterStrategyUp:
		return "Up"
	case FilterStrategyAverage:
		return "Average"
	case FilterStrategyPaeth:
		return "Paeth"
	case FilterStrategyMinSum:
		return "MinSum"
	case FilterStrategyAdaptive:
		return "Adaptive"
	case FilterStrategyAdaptiveFast:
		return "AdaptiveFast"
	case FilterStrategyEntropy:
		return "Entropy"
	case FilterStrategyBruteForce:
		return "BruteForce"
	case FilterStrategyBigrams:
		return "Bigrams"
	case FilterStrategyParallel:
		return "Parallel"
	default:
		return "Unknown"
	}
}

func TestSelectAllParallelDifferentStrategies(t *testing.T) {
	width, height, bpp := 32, 128, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	strategies := []FilterStrategy{
		FilterStrategyMinSum,
		FilterStrategyAdaptive,
		FilterStrategyAdaptiveFast,
		FilterStrategyEntropy,
		FilterStrategySub,
		FilterStrategyUp,
		FilterStrategyAverage,
		FilterStrategyPaeth,
	}

	for _, strategy := range strategies {
		t.Run(strategyName(strategy), func(t *testing.T) {
			config := DefaultParallelConfig()
			config.Strategy = strategy

			sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, strategy)
			parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)

			if len(sequentialFilters) != len(parallelFilters) {
				t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
			}

			for i := range sequentialFilters {
				if sequentialFilters[i] != parallelFilters[i] {
					t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
				}
			}
		})
	}
}

func TestSelectAllParallelBigrams(t *testing.T) {
	width, height, bpp := 32, 128, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	sequentialFilters := SelectAllBigrams(pixels, width, height, bpp)
	parallelFilters := SelectAllParallelBigrams(pixels, width, height, bpp)

	if len(sequentialFilters) != len(parallelFilters) {
		t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
	}

	for i := range sequentialFilters {
		if sequentialFilters[i] != parallelFilters[i] {
			t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
		}
	}
}

func TestSelectAllParallelBigramsSmallImage(t *testing.T) {
	width, height, bpp := 8, 16, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	sequentialFilters := SelectAllBigrams(pixels, width, height, bpp)
	parallelFilters := SelectAllParallelBigrams(pixels, width, height, bpp)

	if len(sequentialFilters) != len(parallelFilters) {
		t.Errorf("filter count mismatch: sequential=%d, parallel=%d", len(sequentialFilters), len(parallelFilters))
	}

	for i := range sequentialFilters {
		if sequentialFilters[i] != parallelFilters[i] {
			t.Errorf("row %d: sequential=%d, parallel=%d", i, sequentialFilters[i], parallelFilters[i])
		}
	}
}

func TestSelectAllParallelThreadSafety(t *testing.T) {
	width, height, bpp := 64, 128, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	results := make([][]FilterType, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = SelectAllParallel(pixels, width, height, bpp)
		}(i)
	}

	wg.Wait()

	sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

	for i := 0; i < numGoroutines; i++ {
		if len(results[i]) != len(sequentialFilters) {
			t.Errorf("goroutine %d: filter count mismatch: expected=%d, got=%d", i, len(sequentialFilters), len(results[i]))
			continue
		}

		for j := range sequentialFilters {
			if results[i][j] != sequentialFilters[j] {
				t.Errorf("goroutine %d, row %d: expected=%d, got=%d", i, j, sequentialFilters[j], results[i][j])
			}
		}
	}
}

func TestSelectAllParallelConfig(t *testing.T) {
	width, height, bpp := 64, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	t.Run("custom workers", func(t *testing.T) {
		config := DefaultParallelConfig()
		config.NumWorkers = 4
		parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
		sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

		if len(parallelFilters) != len(sequentialFilters) {
			t.Errorf("filter count mismatch")
		}

		for i := range sequentialFilters {
			if parallelFilters[i] != sequentialFilters[i] {
				t.Errorf("row %d: expected=%d, got=%d", i, sequentialFilters[i], parallelFilters[i])
			}
		}
	})

	t.Run("custom threshold", func(t *testing.T) {
		config := DefaultParallelConfig()
		config.Threshold = 64
		parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
		sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

		if len(parallelFilters) != len(sequentialFilters) {
			t.Errorf("filter count mismatch")
		}

		for i := range sequentialFilters {
			if parallelFilters[i] != sequentialFilters[i] {
				t.Errorf("row %d: expected=%d, got=%d", i, sequentialFilters[i], parallelFilters[i])
			}
		}
	})

	t.Run("zero workers defaults to NumCPU", func(t *testing.T) {
		config := DefaultParallelConfig()
		config.NumWorkers = 0
		parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
		sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

		if len(parallelFilters) != len(sequentialFilters) {
			t.Errorf("filter count mismatch")
		}

		for i := range sequentialFilters {
			if parallelFilters[i] != sequentialFilters[i] {
				t.Errorf("row %d: expected=%d, got=%d", i, sequentialFilters[i], parallelFilters[i])
			}
		}
	})
}

func TestSelectAllParallelEmpty(t *testing.T) {
	t.Run("zero height", func(t *testing.T) {
		pixels := make([]byte, 100)
		filters := SelectAllParallel(pixels, 10, 0, 4)
		if len(filters) != 0 {
			t.Errorf("expected 0 filters for zero height, got %d", len(filters))
		}
	})

	t.Run("zero width", func(t *testing.T) {
		height := 10
		bpp := 4
		pixels := make([]byte, 0)
		filters := SelectAllParallel(pixels, 0, height, bpp)
		if len(filters) != height {
			t.Errorf("expected %d filters for zero width, got %d", height, len(filters))
		}
	})
}

func BenchmarkSelectAllParallelVsSequential(b *testing.B) {
	sizes := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"small 64x64 RGBA", 64, 64, 4},
		{"medium 128x128 RGBA", 128, 128, 4},
		{"large 256x256 RGBA", 256, 256, 4},
		{"xlarge 512x512 RGBA", 512, 512, 4},
		{"xxlarge 1024x1024 RGBA", 1024, 1024, 4},
	}

	cpus := runtime.NumCPU()
	b.Logf("Number of CPUs: %d", cpus)

	for _, size := range sizes {
		pixels := make([]byte, size.width*size.height*size.bpp)
		for i := range pixels {
			pixels[i] = byte(i % 256)
		}

		b.Run(size.name+"_sequential", func(b *testing.B) {
			b.SetBytes(int64(size.width * size.height * size.bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllWithStrategy(pixels, size.width, size.height, size.bpp, FilterStrategyAdaptive)
			}
		})

		b.Run(size.name+"_parallel", func(b *testing.B) {
			b.SetBytes(int64(size.width * size.height * size.bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllParallel(pixels, size.width, size.height, size.bpp)
			}
		})
	}
}

func BenchmarkSelectAllParallelWithConfig(b *testing.B) {
	sizes := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"medium 128x128 RGBA", 128, 128, 4},
		{"large 256x256 RGBA", 256, 256, 4},
		{"xlarge 512x512 RGBA", 512, 512, 4},
	}

	for _, size := range sizes {
		pixels := make([]byte, size.width*size.height*size.bpp)
		for i := range pixels {
			pixels[i] = byte(i % 256)
		}

		for workers := 1; workers <= runtime.NumCPU(); workers *= 2 {
			b.Run(size.name+"_workers_"+string(rune('0'+workers)), func(b *testing.B) {
				config := DefaultParallelConfig()
				config.NumWorkers = workers

				b.SetBytes(int64(size.width * size.height * size.bpp))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = SelectAllParallelWithConfig(pixels, size.width, size.height, size.bpp, config)
				}
			})
		}
	}
}

func BenchmarkSelectAllParallelBigrams(b *testing.B) {
	sizes := []struct {
		name   string
		width  int
		height int
		bpp    int
	}{
		{"small 64x64 RGBA", 64, 64, 4},
		{"medium 128x128 RGBA", 128, 128, 4},
		{"large 256x256 RGBA", 256, 256, 4},
	}

	for _, size := range sizes {
		pixels := make([]byte, size.width*size.height*size.bpp)
		for i := range pixels {
			pixels[i] = byte(i % 256)
		}

		b.Run(size.name+"_sequential", func(b *testing.B) {
			b.SetBytes(int64(size.width * size.height * size.bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllBigrams(pixels, size.width, size.height, size.bpp)
			}
		})

		b.Run(size.name+"_parallel", func(b *testing.B) {
			b.SetBytes(int64(size.width * size.height * size.bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllParallelBigrams(pixels, size.width, size.height, size.bpp)
			}
		})
	}
}

func BenchmarkSelectAllParallelDifferentStrategies(b *testing.B) {
	width, height, bpp := 256, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	strategies := []FilterStrategy{
		FilterStrategyMinSum,
		FilterStrategyAdaptiveFast,
		FilterStrategyEntropy,
	}

	for _, strategy := range strategies {
		b.Run("sequential_"+strategyName(strategy), func(b *testing.B) {
			b.SetBytes(int64(width * height * bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllWithStrategy(pixels, width, height, bpp, strategy)
			}
		})

		b.Run("parallel_"+strategyName(strategy), func(b *testing.B) {
			config := DefaultParallelConfig()
			config.Strategy = strategy

			b.SetBytes(int64(width * height * bpp))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SelectAllParallelWithConfig(pixels, width, height, bpp, config)
			}
		})
	}
}

func TestSelectAllWithStrategyParallel(t *testing.T) {
	width, height, bpp := 64, 128, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	parallelFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyParallel)
	referenceFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

	if len(parallelFilters) != len(referenceFilters) {
		t.Errorf("filter count mismatch: parallel=%d, reference=%d", len(parallelFilters), len(referenceFilters))
	}

	for i := range parallelFilters {
		if parallelFilters[i] != referenceFilters[i] {
			t.Errorf("row %d: parallel=%d, reference=%d", i, parallelFilters[i], referenceFilters[i])
		}
	}
}

func TestSelectAllWithStrategyParallelSmallImage(t *testing.T) {
	width, height, bpp := 8, 16, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	parallelFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyParallel)
	referenceFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

	if len(parallelFilters) != len(referenceFilters) {
		t.Errorf("filter count mismatch: parallel=%d, reference=%d", len(parallelFilters), len(referenceFilters))
	}

	for i := range parallelFilters {
		if parallelFilters[i] != referenceFilters[i] {
			t.Errorf("row %d: parallel=%d, reference=%d", i, parallelFilters[i], referenceFilters[i])
		}
	}
}

func TestSelectAllWithStrategyParallelDifferentStrategies(t *testing.T) {
	testStrategies := []FilterStrategy{
		FilterStrategyMinSum,
		FilterStrategyAdaptiveFast,
		FilterStrategyEntropy,
		FilterStrategySub,
		FilterStrategyUp,
		FilterStrategyAverage,
		FilterStrategyPaeth,
	}

	width, height, bpp := 128, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	for _, strategy := range testStrategies {
		t.Run(strategyName(strategy), func(t *testing.T) {
			config := DefaultParallelConfig()
			config.Strategy = strategy

			parallelViaConfig := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
			sequentialViaStrategy := SelectAllWithStrategy(pixels, width, height, bpp, strategy)

			if len(parallelViaConfig) != len(sequentialViaStrategy) {
				t.Errorf("filter count mismatch: config=%d, strategy=%d", len(parallelViaConfig), len(sequentialViaStrategy))
			}

			for i := range parallelViaConfig {
				if parallelViaConfig[i] != sequentialViaStrategy[i] {
					t.Errorf("row %d: config=%d, strategy=%d", i, parallelViaConfig[i], sequentialViaStrategy[i])
				}
			}
		})
	}
}

func TestAutoThresholdBehavior(t *testing.T) {
	width, bpp := 64, 4

	testCases := []struct {
		name   string
		height int
	}{
		{"exactly at threshold", 32},
		{"just above threshold", 33},
		{"small image", 16},
		{"medium image", 64},
		{"large image", 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			height := tc.height
			pixels := make([]byte, width*height*bpp)
			for i := range pixels {
				pixels[i] = byte(i % 256)
			}

			parallelFilters := SelectAllParallel(pixels, width, height, bpp)
			sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

			if len(parallelFilters) != len(sequentialFilters) {
				t.Errorf("filter count mismatch: parallel=%d, sequential=%d", len(parallelFilters), len(sequentialFilters))
			}

			for i := range parallelFilters {
				if parallelFilters[i] != sequentialFilters[i] {
					t.Errorf("row %d: parallel=%d, sequential=%d", i, parallelFilters[i], sequentialFilters[i])
				}
			}
		})
	}
}

func TestAutoThresholdWithCustomConfig(t *testing.T) {
	width, bpp := 64, 4
	height := 64
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	t.Run("custom threshold 100", func(t *testing.T) {
		config := DefaultParallelConfig()
		config.Threshold = 100

		parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
		sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

		if len(parallelFilters) != len(sequentialFilters) {
			t.Errorf("filter count mismatch")
		}

		for i := range parallelFilters {
			if parallelFilters[i] != sequentialFilters[i] {
				t.Errorf("row %d: parallel=%d, sequential=%d", i, parallelFilters[i], sequentialFilters[i])
			}
		}
	})

	t.Run("custom threshold 32", func(t *testing.T) {
		config := DefaultParallelConfig()
		config.Threshold = 32

		parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
		sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, FilterStrategyAdaptive)

		if len(parallelFilters) != len(sequentialFilters) {
			t.Errorf("filter count mismatch")
		}

		for i := range parallelFilters {
			if parallelFilters[i] != sequentialFilters[i] {
				t.Errorf("row %d: parallel=%d, sequential=%d", i, parallelFilters[i], sequentialFilters[i])
			}
		}
	})
}

func TestParallelStrategyConsistency(t *testing.T) {
	width, height, bpp := 128, 256, 4
	pixels := make([]byte, width*height*bpp)
	for i := range pixels {
		pixels[i] = byte(i % 256)
	}

	strategies := []FilterStrategy{
		FilterStrategyAdaptiveFast,
		FilterStrategyMinSum,
	}

	for _, strategy := range strategies {
		t.Run(strategyName(strategy), func(t *testing.T) {
			config := DefaultParallelConfig()
			config.Strategy = strategy

			parallelFilters := SelectAllParallelWithConfig(pixels, width, height, bpp, config)
			sequentialFilters := SelectAllWithStrategy(pixels, width, height, bpp, strategy)

			if len(parallelFilters) != len(sequentialFilters) {
				t.Errorf("filter count mismatch")
			}

			for i := range parallelFilters {
				if parallelFilters[i] != sequentialFilters[i] {
					t.Errorf("row %d: parallel=%d, sequential=%d", i, parallelFilters[i], sequentialFilters[i])
				}
			}
		})
	}
}
