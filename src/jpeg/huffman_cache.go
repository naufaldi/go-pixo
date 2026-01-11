package jpeg

import (
	"sync"
)

type CacheKey struct {
	Quality     uint8
	Subsampling string
	Optimized   bool
}

type HuffmanCache struct {
	mu        sync.RWMutex
	tables    map[CacheKey]*HuffmanTables
	hitCount  int64
	missCount int64
}

func NewHuffmanCache() *HuffmanCache {
	return &HuffmanCache{
		tables: make(map[CacheKey]*HuffmanTables),
	}
}

func (c *HuffmanCache) GetHuffmanTables(quality uint8, subsampling Subsampling, optimized bool) *HuffmanTables {
	subsamplingStr := "444"
	if subsampling == Subsampling420 {
		subsamplingStr = "420"
	}

	key := CacheKey{
		Quality:     quality,
		Subsampling: subsamplingStr,
		Optimized:   optimized,
	}

	c.mu.RLock()
	tables, exists := c.tables[key]
	if exists {
		c.mu.RUnlock()
		c.mu.Lock()
		c.hitCount++
		c.mu.Unlock()
		return tables
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	tables, exists = c.tables[key]
	if exists {
		c.hitCount++
		return tables
	}

	c.missCount++

	if optimized {
		tables = NewHuffmanTables()
	} else {
		tables = NewHuffmanTables()
	}

	c.tables[key] = tables
	return tables
}

func (c *HuffmanCache) Prewarm() {
	qualities := []uint8{50, 75, 90}
	subsamplings := []Subsampling{Subsampling444, Subsampling420}

	for _, quality := range qualities {
		for _, subsampling := range subsamplings {
			for _, optimized := range []bool{false, true} {
				key := CacheKey{
					Quality:     quality,
					Subsampling: subsamplingString(subsampling),
					Optimized:   optimized,
				}
				if _, exists := c.tables[key]; !exists {
					tables := NewHuffmanTables()
					c.tables[key] = tables
				}
			}
		}
	}
}

func subsamplingString(s Subsampling) string {
	if s == Subsampling420 {
		return "420"
	}
	return "444"
}

func (c *HuffmanCache) HitRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	total := c.hitCount + c.missCount
	if total == 0 {
		return 0
	}
	return float64(c.hitCount) / float64(total) * 100
}

func (c *HuffmanCache) Stats() (hits, misses int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hitCount, c.missCount
}

func (c *HuffmanCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.tables)
}

func (c *HuffmanCache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hitCount = 0
	c.missCount = 0
}
