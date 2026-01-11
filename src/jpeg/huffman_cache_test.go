package jpeg

import (
	"sync"
	"testing"
)

func TestHuffmanCacheBasic(t *testing.T) {
	cache := NewHuffmanCache()

	t.Run("cache miss on first access", func(t *testing.T) {
		tables := cache.GetHuffmanTables(75, Subsampling444, false)
		if tables == nil {
			t.Fatal("expected non-nil tables")
		}
		hits, misses := cache.Stats()
		if hits != 0 {
			t.Errorf("expected 0 hits, got %d", hits)
		}
		if misses != 1 {
			t.Errorf("expected 1 miss, got %d", misses)
		}
	})

	t.Run("cache hit on second access", func(t *testing.T) {
		tables1 := cache.GetHuffmanTables(75, Subsampling444, false)
		tables2 := cache.GetHuffmanTables(75, Subsampling444, false)
		if tables1 != tables2 {
			t.Error("expected same table instance on cache hit")
		}
		hits, misses := cache.Stats()
		if hits != 2 {
			t.Errorf("expected 2 hits (cumulative from both subtests), got %d", hits)
		}
		if misses != 1 {
			t.Errorf("expected 1 miss, got %d", misses)
		}
	})

	t.Run("different keys produce different tables", func(t *testing.T) {
		tables1 := cache.GetHuffmanTables(75, Subsampling444, false)
		tables2 := cache.GetHuffmanTables(85, Subsampling444, false)
		if tables1 == tables2 {
			t.Error("expected different table instances for different quality")
		}

		tables3 := cache.GetHuffmanTables(75, Subsampling420, false)
		if tables1 == tables3 {
			t.Error("expected different table instances for different subsampling")
		}

		tables4 := cache.GetHuffmanTables(75, Subsampling444, true)
		if tables1 == tables4 {
			t.Error("expected different table instances for optimized flag")
		}
	})

	t.Run("cache hit rate is tracked", func(t *testing.T) {
		cache.ResetStats()
		_ = cache.GetHuffmanTables(75, Subsampling444, false)
		_ = cache.GetHuffmanTables(75, Subsampling444, false)
		_ = cache.GetHuffmanTables(75, Subsampling444, false)
		rate := cache.HitRate()
		if rate < 66 {
			t.Errorf("expected hit rate > 66%% (2 hits out of 3), got %.2f%%", rate)
		}
	})
}

func TestHuffmanCachePrewarm(t *testing.T) {
	cache := NewHuffmanCache()
	cache.Prewarm()

	expectedKeys := 12 // 3 qualities × 2 subsamplings × 2 optimized variants
	if cache.Len() != expectedKeys {
		t.Errorf("expected %d entries after prewarm, got %d", expectedKeys, cache.Len())
	}

	t.Run("prewarmed entries are cache hits", func(t *testing.T) {
		cache.ResetStats()
		for _, quality := range []uint8{50, 75, 90} {
			for _, subsampling := range []Subsampling{Subsampling444, Subsampling420} {
				for _, optimized := range []bool{false, true} {
					_ = cache.GetHuffmanTables(quality, subsampling, optimized)
				}
			}
		}
		hits, misses := cache.Stats()
		if hits != int64(expectedKeys) {
			t.Errorf("expected %d hits, got %d", expectedKeys, hits)
		}
		if misses != 0 {
			t.Errorf("expected 0 misses, got %d", misses)
		}
	})
}

func TestHuffmanCacheThreadSafety(t *testing.T) {
	cache := NewHuffmanCache()
	cache.Prewarm()

	var wg sync.WaitGroup
	numGoroutines := 100
	numAccesses := 1000

	t.Run("concurrent reads", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numAccesses; j++ {
					quality := uint8((j % 100) + 1)
					subsampling := Subsampling(j % 2)
					optimized := j%2 == 0
					_ = cache.GetHuffmanTables(quality, subsampling, optimized)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("concurrent access with stats", func(t *testing.T) {
		cache.ResetStats()
		var mu sync.Mutex
		var hitSum, missSum int64

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numAccesses; j++ {
					quality := uint8((j % 100) + 1)
					subsampling := Subsampling(j % 2)
					optimized := j%2 == 0
					_ = cache.GetHuffmanTables(quality, subsampling, optimized)

					hits, misses := cache.Stats()
					mu.Lock()
					hitSum = hits
					missSum = misses
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()

		total := hitSum + missSum
		if total != int64(numGoroutines*numAccesses) {
			t.Errorf("expected %d total accesses, got %d", numGoroutines*numAccesses, total)
		}
	})
}

func TestHuffmanCacheHitRate(t *testing.T) {
	cache := NewHuffmanCache()
	cache.Prewarm()

	cache.ResetStats()

	patterns := []struct {
		quality     uint8
		subsampling Subsampling
		optimized   bool
		count       int
	}{
		{75, Subsampling444, false, 100},
		{75, Subsampling444, false, 200},
		{90, Subsampling420, true, 50},
		{50, Subsampling420, false, 30},
	}

	for _, p := range patterns {
		for i := 0; i < p.count; i++ {
			_ = cache.GetHuffmanTables(p.quality, p.subsampling, p.optimized)
		}
	}

	hitRate := cache.HitRate()
	t.Logf("Cache hit rate: %.2f%%", hitRate)

	if hitRate < 80 {
		t.Errorf("expected hit rate > 80%%, got %.2f%%", hitRate)
	}
}

func TestHuffmanCacheMultipleQualityLevels(t *testing.T) {
	cache := NewHuffmanCache()

	qualities := []uint8{10, 25, 50, 75, 90, 100}
	subsamplings := []Subsampling{Subsampling444, Subsampling420}
	optimizedOptions := []bool{false, true}

	for _, quality := range qualities {
		for _, subsampling := range subsamplings {
			for _, optimized := range optimizedOptions {
				tables := cache.GetHuffmanTables(quality, subsampling, optimized)
				if tables == nil {
					t.Errorf("failed to get tables for quality=%d, subsampling=%v, optimized=%v",
						quality, subsampling, optimized)
				}
			}
		}
	}

	if cache.Len() != len(qualities)*len(subsamplings)*len(optimizedOptions) {
		t.Errorf("expected %d entries, got %d",
			len(qualities)*len(subsamplings)*len(optimizedOptions), cache.Len())
	}
}

func BenchmarkHuffmanCache(b *testing.B) {
	cache := NewHuffmanCache()
	cache.Prewarm()

	b.Run("cache hit", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cache.GetHuffmanTables(75, Subsampling444, false)
		}
	})

	b.Run("cache miss", func(b *testing.B) {
		cache2 := NewHuffmanCache()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			quality := uint8((i % 100) + 1)
			subsampling := Subsampling(i % 2)
			optimized := i%2 == 0
			_ = cache2.GetHuffmanTables(quality, subsampling, optimized)
		}
	})

	b.Run("concurrent access", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = cache.GetHuffmanTables(75, Subsampling444, false)
			}
		})
	})
}

func TestHuffmanCacheSubsamplingString(t *testing.T) {
	tests := []struct {
		subsampling Subsampling
		expected    string
	}{
		{Subsampling444, "444"},
		{Subsampling420, "420"},
	}

	for _, tt := range tests {
		result := subsamplingString(tt.subsampling)
		if result != tt.expected {
			t.Errorf("subsamplingString(%v) = %s, want %s", tt.subsampling, result, tt.expected)
		}
	}
}

func TestHuffmanCacheStatsReset(t *testing.T) {
	cache := NewHuffmanCache()
	_ = cache.GetHuffmanTables(75, Subsampling444, false)
	_ = cache.GetHuffmanTables(75, Subsampling444, false)

	if cache.HitRate() != 50 {
		t.Errorf("expected 50%% hit rate before reset, got %.2f%%", cache.HitRate())
	}

	cache.ResetStats()

	hits, misses := cache.Stats()
	if hits != 0 || misses != 0 {
		t.Errorf("expected 0,0 after reset, got %d,%d", hits, misses)
	}
}

func TestHuffmanCacheIntegration(t *testing.T) {
	privateCache := NewHuffmanCache()

	t.Run("standard tables use cache", func(t *testing.T) {
		tables1 := privateCache.GetHuffmanTables(75, Subsampling444, false)
		tables2 := privateCache.GetHuffmanTables(75, Subsampling444, false)

		if tables1 != tables2 {
			t.Error("expected same table instance for same parameters")
		}

		hits, misses := privateCache.Stats()
		if misses != 1 {
			t.Errorf("expected 1 miss on first access, got %d", misses)
		}
		if hits != 1 {
			t.Errorf("expected 1 hit on second access, got %d", hits)
		}
	})

	t.Run("different subsampling uses different cache entries", func(t *testing.T) {
		tables420 := privateCache.GetHuffmanTables(75, Subsampling420, false)
		tables444 := privateCache.GetHuffmanTables(75, Subsampling444, false)

		if tables420 == tables444 {
			t.Error("expected different tables for different subsampling")
		}
	})

	t.Run("cache hit rate improves with repeated access", func(t *testing.T) {
		privateCache.ResetStats()

		for i := 0; i < 10; i++ {
			_ = privateCache.GetHuffmanTables(75, Subsampling444, false)
		}

		hitRate := privateCache.HitRate()
		if hitRate < 80 {
			t.Errorf("expected hit rate > 80%%, got %.2f%%", hitRate)
		}
	})
}

func BenchmarkHuffmanCache_VsDirect(b *testing.B) {
	width, height := 256, 256
	pixels := make([]byte, width*height*3)
	for i := range pixels {
		pixels[i] = uint8(i % 256)
	}

	b.Run("direct NewHuffmanTables", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewHuffmanTables()
		}
	})

	b.Run("cached GetStandardHuffmanTables", func(b *testing.B) {
		huffmanCache.ResetStats()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = GetStandardHuffmanTables(75, Subsampling444)
		}
	})
}
