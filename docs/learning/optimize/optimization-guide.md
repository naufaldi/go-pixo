# Optimization Guide for Go-pixo

Actionable recommendations to optimize the Go implementation based on Rust comparison analysis.

## Table of Contents

1. [Overview](#1-overview)
2. [Priority 1: High-Impact Optimizations](#2-priority-1-high-impact-optimizations)
3. [Priority 2: Medium-Impact Optimizations](#3-priority-2-medium-impact-optimizations)
4. [Priority 3: Low-Impact Optimizations](#4-priority-3-low-impact-low-effort)
5. [Related Documents](#5-related-documents)

---

## 1. Overview

This guide provides actionable optimization recommendations for go-pixo based on the detailed comparison with the Rust implementation (pixo). The recommendations are categorized by impact and effort, allowing you to prioritize work effectively.

The optimizations are organized into three priority levels:
- **Priority 1**: High impact, medium effort - Quick wins with significant improvements
- **Priority 2**: Medium impact, high effort - Significant improvements requiring more work
- **Priority 3**: Low impact, low effort - Incremental improvements for polish

---

## 2. Priority 1: High-Impact Optimizations

### 2.1 Add Palette Lookup Table (LUT) for PNG Quantization

**Impact**: 10-100x speedup for quantization phase
**Effort**: Medium
**Related files**: `go-pixo/src/png/quantize.go`, `go-pixo/src/png/palette.go`

#### Problem

Currently, Go uses linear search O(n) per pixel for palette indexing. For a 256-color palette, this means up to 256 distance calculations per pixel, which is extremely slow for large images.

#### Solution

Implement a 6-6-6 RGB lookup table for O(1) palette indexing.

```go
package png

// PaletteLut provides O(1) palette lookup using a precomputed table.
type PaletteLut struct {
    opaqueLut [64][64][64]uint8
    palette   Palette
}

// NewPaletteLut creates a new palette lookup table.
func NewPaletteLut(palette Palette) *PaletteLut {
    lut := &PaletteLut{palette: palette}
    for r6 := 0; r6 < 64; r6++ {
        for g6 := 0; g6 < 64; g6++ {
            for b6 := 0; b6 < 64; b6++ {
                r8 := (r6 << 2) | (r6 >> 4)
                g8 := (g6 << 2) | (g6 >> 4)
                b8 := (b6 << 2) | (b6 >> 4)
                lut.opaqueLut[r6][g6][b6] = uint8(palette.FindNearestIndex(r8, g8, b8))
            }
        }
    }
    return lut
}

// Lookup returns the palette index for an opaque RGB color in O(1) time.
func (lut *PaletteLut) Lookup(r, g, b uint8) uint8 {
    r6 := r >> 2
    g6 := g >> 2
    b6 := b >> 2
    return lut.opaqueLut[r6][g6][b6]
}

// FindNearestIndex finds the nearest palette color using the LUT.
func (p *Palette) FindNearestIndex(r, g, b uint8) uint8 {
    lut := NewPaletteLut(*p)
    return lut.Lookup(r, g, b)
}
```

#### Implementation Steps

1. Create the `PaletteLut` struct with 64x64x64 array
2. Precompute all 262,144 entries at initialization
3. Modify `Quantize` function to use LUT lookup
4. Fall back to linear search for transparent pixels

#### Expected Results

- **Quantization speed**: 10-100x improvement
- **Memory usage**: +256KB for LUT
- **Quality**: No change (identical results)

### 2.2 Add K-means Palette Refinement

**Impact**: 5-15% better visual quality for quantized images
**Effort**: Medium
**Related files**: `go-pixo/src/png/quantize.go`, `go-pixo/src/png/median_cut.go`

#### Problem

Median-cut produces palette colors based only on color space distribution, not on actual image content. This can lead to suboptimal palettes for photographic images.

#### Solution

Add 2-3 iterations of K-means refinement after median-cut to adjust palette colors to match actual image data.

```go
package png

// RefinePaletteKmeans applies K-means refinement to a palette.
func RefinePaletteKmeans(palette *Palette, colors []ColorCount, iterations int) {
    for iter := 0; iter < iterations; iter++ {
        // Accumulate weighted centroids
        accumulators := make([][5]int, palette.Len())
        for _, c := range colors {
            bestIdx := palette.FindNearestIndex(c.Color.R, c.Color.G, c.Color.B)
            acc := &accumulators[bestIdx]
            acc[0] += int(c.Color.R) * c.Count
            acc[1] += int(c.Color.G) * c.Count
            acc[2] += int(c.Color.B) * c.Count
            acc[4] += c.Count
        }

        // Update palette to centroids
        for i, acc := range accumulators {
            if acc[4] > 0 {
                palette.Colors[i] = Color{
                    R: uint8(acc[0] / acc[4]),
                    G: uint8(acc[1] / acc[4]),
                    B: uint8(acc[2] / acc[4]),
                }
            }
        }
    }
}
```

#### Implementation Steps

1. Create `RefinePaletteKmeans` function
2. Modify `Quantize` to call refinement after median-cut
3. Use 2-3 iterations for balance of quality and speed
4. Update tests to verify quality improvement

#### Expected Results

- **Visual quality**: 5-15% improvement for photographic content
- **Speed**: ~10% slowdown (negligible compared to LUT speedup)
- **Compression**: 2-5% better for quantized images

### 2.3 Implement Bigrams Filter Strategy

**Impact**: 2-5% better compression ratio
**Effort**: Medium
**Related files**: `go-pixo/src/png/filter_selector.go`, `go-pixo/src/png/filter_*.go`

#### Problem

Go-pixo lacks the `Bigrams` filter strategy that optimizes for DEFLATE LZ77 matching by minimizing distinct byte pairs.

#### Solution

Add a new filter strategy that counts and minimizes distinct bigrams (byte pairs) in filtered output.

```go
package png

// FilterStrategyBigrams selects the filter that minimizes distinct bigrams.
const FilterStrategyBigrams FilterStrategy = "Bigrams"

func selectBigrams(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    var bestFilter FilterType
    var bestFiltered []byte
    bestBigrams := -1

    filters := []struct {
        typ FilterType
        fn  func() []byte
    }{
        {FilterNone, func() []byte { return ApplyFilterNone(row) }},
        {FilterSub, func() []byte { return ApplyFilterSub(row, bpp) }},
        {FilterUp, func() []byte { return ApplyFilterUp(row, prevRow) }},
        {FilterAverage, func() []byte { return ApplyFilterAverage(row, prevRow, bpp) }},
        {FilterPaeth, func() []byte { return ApplyFilterPaeth(row, prevRow, bpp) }},
    }

    for _, f := range filters {
        filtered := f.fn()
        bigrams := countDistinctBigrams(filtered)
        if bestBigrams < 0 || bigrams < bestBigrams {
            bestBigrams = bigrams
            bestFilter = f.typ
            bestFiltered = filtered
        }
    }

    return bestFilter, bestFiltered
}

func countDistinctBigrams(data []byte) int {
    if len(data) < 2 {
        return 0
    }
    bigrams := make(map[uint16]bool)
    for i := 0; i < len(data)-1; i++ {
        bigram := uint16(data[i])<<8 | uint16(data[i+1])
        bigrams[bigram] = true
    }
    return len(bigrams)
}
```

#### Implementation Steps

1. Add `FilterStrategyBigrams` constant
2. Implement `selectBigrams` function
3. Update `SelectFilterWithStrategy` to handle new strategy
4. Add tests comparing compression ratio

#### Expected Results

- **Compression ratio**: 2-5% improvement on typical images
- **Speed**: Similar to MinSum strategy
- **Compatibility**: No breaking changes

### 2.4 Add SIMD Acceleration for DCT

**Impact**: 3-5x speedup for JPEG DCT operations
**Effort**: Medium-High
**Related files**: `go-pixo/src/jpeg/dct.go`

#### Problem

Go-pixo uses pure floating-point DCT without SIMD acceleration, while pixo uses SIMD-accelerated integer DCT.

#### Solution

Use Go's assembly capabilities or golang.org/x/exp/simd for DCT acceleration.

```go
package jpeg

import "golang.org/x/exp/simd"

// ForwardDCTSIMD performs DCT using SIMD instructions.
func ForwardDCTSIMD(block [64]float64) [64]float64 {
    // Use SIMD for horizontal transform
    // Fall back to scalar for vertical transform
    var result [64]float64

    // SIMD-accelerated horizontal DCT
    simdBlock := simd.Load64(&block)
    // ... SIMD operations ...

    return result
}
```

#### Implementation Steps

1. Add golang.org/x/exp/simd dependency
2. Create SIMD version of ForwardDCT
3. Add runtime feature detection
4. Fall back to scalar on unsupported platforms
5. Benchmark to verify improvement

#### Expected Results

- **DCT speed**: 3-5x improvement
- **JPEG encoding**: 20-30% overall speedup
- **Compatibility**: Graceful fallback on old CPUs

### 2.5 Optimize Huffman Table Generation

**Impact**: 10-20% faster JPEG encoding
**Effort**: Medium
**Related files**: `go-pixo/src/jpeg/huffman.go`, `go-pixo/src/jpeg/huffman_optimized.go`

#### Problem

Huffman tables are computed fresh for each image, even though many images share similar statistics.

#### Solution

Precompute and cache Huffman tables for common quantization matrices and quality levels.

```go
package jpeg

import "sync"

// HuffmanCache provides cached Huffman tables for common configurations.
var huffmanCache struct {
    sync.RWMutex
    tables map[huffmanCacheKey]*HuffmanTables
}

type huffmanCacheKey struct {
    quality    int
    subsampling string
}

// GetHuffmanTables returns cached or newly computed Huffman tables.
func GetHuffmanTables(quality int, subsampling string) *HuffmanTables {
    key := huffmanCacheKey{quality, subsampling}

    huffmanCache.RLock()
    if tables, ok := huffmanCache.tables[key]; ok {
        huffmanCache.RUnlock()
        return tables
    }
    huffmanCache.RUnlock()

    // Compute new tables
    tables := ComputeHuffmanTables(quality, subsampling)

    huffmanCache.Lock()
    if existing, ok := huffmanCache.tables[key]; ok {
        huffmanCache.Unlock()
        return existing
    }
    huffmanCache.tables[key] = tables
    huffmanCache.Unlock()

    return tables
}
```

#### Implementation Steps

1. Create cache data structure with mutex
2. Add `GetHuffmanTables` function with cache lookup
3. Pre-populate cache for common quality levels (50, 75, 90)
4. Add cache warming at startup
5. Monitor cache hit rate

#### Expected Results

- **Huffman generation**: 80-90% cache hit rate typical
- **JPEG encoding**: 10-20% faster on similar images
- **Memory**: ~100KB for cache

---

## 3. Priority 2: Medium-Impact Optimizations

### 3.1 Add Parallel Filter Selection for PNG

**Impact**: 2-8x speedup (proportional to CPU cores)
**Effort**: High
**Related files**: `go-pixo/src/png/filter_selector.go`

#### Solution

Use goroutines to process independent rows in parallel.

```go
package png

func SelectAllParallel(pixels []byte, width, height, bpp int) []FilterType {
    if height <= 32 {
        // Not worth the overhead for small images
        return SelectAll(pixels, width, height, bpp)
    }

    filters := make([]FilterType, height)
    rowBytes := width * bpp

    // Process rows in parallel
    type result struct {
        y      int
        filter FilterType
    }
    results := make(chan result, height)

    for y := 0; y < height; y++ {
        go func(y int) {
            offset := y * rowBytes
            row := pixels[offset : offset+rowBytes]
            var prevRow []byte
            if y > 0 {
                prevOffset := (y - 1) * rowBytes
                prevRow = pixels[prevOffset : prevOffset+rowBytes]
            }
            filter, _ := SelectFilter(row, prevRow, bpp)
            results <- result{y, filter}
        }(y)
    }

    // Collect results
    for i := 0; i < height; i++ {
        r := <-results
        filters[r.y] = r.filter
    }

    return filters
}
```

### 3.2 Implement Full Trellis Optimization for JPEG

**Impact**: 5-10% better quality or 10-20% smaller files
**Effort**: High
**Related files**: `go-pixo/src/jpeg/trellis.go`

#### Solution

Expand trellis optimization with complete dynamic programming, perceptual distortion metrics, and accurate rate estimation.

### 3.3 Add Progressive Scan Streaming

**Impact**: 50% memory reduction, better latency
**Effort**: Medium
**Related files**: `go-pixo/src/jpeg/progressive.go`

#### Solution

Refactor progressive encoder to stream data rather than buffering entire image.

### 3.4 Implement Cost-Model Based Optimal Parsing for DEFLATE

**Impact**: 3-8% better compression ratio
**Effort**: High
**Related files**: `go-pixo/src/compress/lz77_encoder.go`, `go-pixo/src/compress/deflate_encoder.go`

#### Solution

Add optimal LZ77 parsing with cost model, similar to Rust's implementation.

---

## 4. Priority 3: Low-Impact Optimizations

### 4.1 Add Adaptive Scratch Buffers

**Impact**: Reduced memory allocation, lower GC overhead
**Effort**: Low
**Related files**: `go-pixo/src/png/filter_selector.go`

#### Solution

Reuse buffers across filter evaluations to reduce GC pressure.

```go
type AdaptiveScratch struct {
    none  []byte
    sub   []byte
    up    []byte
    avg   []byte
    paeth []byte
}

func NewAdaptiveScratch(rowLen int) *AdaptiveScratch {
    return &AdaptiveScratch{
        none:  make([]byte, rowLen),
        sub:   make([]byte, rowLen),
        up:    make([]byte, rowLen),
        avg:   make([]byte, rowLen),
        paeth: make([]byte, rowLen),
    }
}
```

### 4.2 Implement Early Termination in Filter Selection

**Impact**: 10-30% faster filter selection
**Effort**: Low
**Related files**: `go-pixo/src/png/filter_selector.go`

#### Solution

Add early_stop threshold to skip remaining filters when best_score is already optimal.

```go
func selectMinSum(row []byte, prevRow []byte, bpp int) (FilterType, []byte) {
    // ... setup ...

    earlyStop := len(row) / 4

    // After evaluating each filter:
    if bestScore <= earlyStop {
        return bestFilter, bestFiltered
    }

    // ... continue ...
}
```

### 4.3 Switch to Redmean Perceptual Distance

**Impact**: Better visual quality for skin tones and gradients
**Effort**: Low
**Related files**: `go-pixo/src/png/quantize.go`

#### Solution

Replace Euclidean palette lookup with Redmean perceptual formula.

```go
func perceptualDistanceSq(c1, c2 Color) uint32 {
    dr := int(c1.R) - int(c2.R)
    dg := int(c1.G) - int(c2.G)
    db := int(c1.B) - int(c2.B)
    da := int(c1.A) - int(c2.A)

    rMean := (int(c1.R) + int(c2.R)) >> 1

    rWeight := 512 + rMean
    bWeight := 767 - rMean
    const gWeight = 1024

    dist := (rWeight*dr*dr + gWeight*dg*dg + bWeight*db*db) >> 8
    return uint32(dist + da*da)
}
```

---

## 5. Related Documents

- [Main Overview](../diff-rust-go.md) - Complete Go vs Rust comparison
- [PNG Comparison](./diff-png.md) - Detailed PNG implementation comparison
- [JPEG Comparison](./diff-jpeg.md) - Detailed JPEG implementation comparison
- [Research Papers](./research-papers.md) - Research papers and references

---

## Implementation Roadmap

### Phase 1: Quick Wins (Week 1)
- [ ] Implement Palette LUT (2.1)
- [ ] Implement K-means refinement (2.2)
- [ ] Implement Bigrams filter (2.3)

### Phase 2: Performance (Week 2-3)
- [ ] Add SIMD for DCT (2.4)
- [ ] Add Huffman caching (2.5)
- [ ] Add parallel filter selection (3.1)

### Phase 3: Quality (Week 3-4)
- [ ] Implement full trellis (3.2)
- [ ] Add progressive streaming (3.3)
- [ ] Implement cost-model parsing (3.4)

### Phase 4: Polish (Week 4-5)
- [ ] Add scratch buffers (4.1)
- [ ] Add early termination (4.2)
- [ ] Switch to Redmean distance (4.3)